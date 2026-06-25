package agent

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/goteamtim/context-monster-cli/internal/ollama"
	"github.com/goteamtim/context-monster-cli/internal/skills"
	"github.com/goteamtim/context-monster-cli/internal/training"
)

// chatClient is the narrow interface Agent uses to communicate with the model.
// *ollama.Client satisfies it automatically; tests can provide a stub.
type chatClient interface {
	Chat(ctx context.Context, messages []ollama.Message, tools []ollama.Tool) (*ollama.ChatResponse, error)
}

// Agent orchestrates the REPL loop and multi-turn tool-calling conversation.
type Agent struct {
	client       chatClient
	skills       []skills.Skill
	systemPrompt string
	history      []ollama.Message
	tools        []ollama.Tool
	verbose      bool
	// allowedPaths restricts which file/dir paths skills may access.
	// An empty slice means unrestricted (backward-compatible default).
	allowedPaths []string
	// logger is non-nil when trajectory recording is enabled.
	logger *training.Logger
	// meta holds static trajectory metadata populated at construction time.
	meta training.TrajectoryMetadata
}

// New creates an Agent wired with the given Ollama client and loaded skills.
// systemPrompt is injected as the first message in the conversation history.
// allowedPaths restricts file/dir access for all tool calls; nil means unrestricted.
// logger may be nil to disable trajectory recording. meta provides static metadata
// (model name, persona, context window, etc.) stamped on every recorded trajectory.
func New(client chatClient, loadedSkills []skills.Skill, systemPrompt string, verbose bool, allowedPaths []string, logger *training.Logger, meta training.TrajectoryMetadata) *Agent {
	tools := make([]ollama.Tool, len(loadedSkills))
	for i, s := range loadedSkills {
		tools[i] = skillToTool(s)
	}

	return &Agent{
		client:       client,
		skills:       loadedSkills,
		systemPrompt: systemPrompt,
		tools:        tools,
		verbose:      verbose,
		allowedPaths: allowedPaths,
		logger:       logger,
		meta:         meta,
		history: []ollama.Message{
			{Role: "system", Content: systemPrompt},
		},
	}
}

// skillToTool converts a Skill into the Tool format expected by the Ollama API.
// This conversion lives here, at the point where skills and ollama types are composed.
func skillToTool(s skills.Skill) ollama.Tool {
	props := make(map[string]ollama.ToolFunctionParam, len(s.Manifest.Parameters.Properties))
	for k, v := range s.Manifest.Parameters.Properties {
		props[k] = ollama.ToolFunctionParam{
			Type:        v.Type,
			Description: v.Description,
		}
	}
	return ollama.Tool{
		Type: "function",
		Function: ollama.ToolFunction{
			Name:        s.Manifest.Name,
			Description: s.Manifest.Description,
			Parameters: ollama.ToolFunctionParameters{
				Type:       s.Manifest.Parameters.Type,
				Properties: props,
				Required:   s.Manifest.Parameters.Required,
			},
		},
	}
}

// Run starts the interactive REPL. It reads lines from stdin, drives the
// multi-turn conversation, and writes assistant replies to stdout.
func (a *Agent) Run(ctx context.Context) {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Context Monster ready. Type your message, or /help for commands.")
	fmt.Println()

	for {
		fmt.Print("You: ")
		if !scanner.Scan() {
			fmt.Println()
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		if handled, keepRunning := a.handleSlashCommand(input); handled {
			if !keepRunning {
				fmt.Println("Bye.")
				break
			}
			continue
		}

		a.history = append(a.history, ollama.Message{
			Role:    "user",
			Content: input,
		})

		startedAt := time.Now()
		reply, messages, inputTokens, outputTokens, err := a.think(ctx, input)
		if err != nil {
			fmt.Fprintf(os.Stderr, "\nError: %v\n\n", err)
			// Remove the user message so history stays consistent on error
			a.history = a.history[:len(a.history)-1]
			continue
		}

		if a.logger != nil {
			meta := a.meta
			meta.ID = training.NewTrajectoryID()
			meta.InputTokens = inputTokens
			meta.OutputTokens = outputTokens
			meta.TotalTokens = inputTokens + outputTokens
			meta.StartedAt = startedAt
			meta.CompletedAt = time.Now()

			systemMsg := training.TrajectoryMessage{Role: "system", Content: a.systemPrompt}
			traj := training.Trajectory{
				Messages: append([]training.TrajectoryMessage{systemMsg}, messages...),
				Metadata: meta,
			}
			if logErr := a.logger.Append(traj); logErr != nil {
				fmt.Fprintf(os.Stderr, "[warning] could not write trajectory: %v\n", logErr)
			}
		}

		fmt.Printf("\nAssistant: %s\n\n", reply)
	}
}

func (a *Agent) handleSlashCommand(input string) (handled bool, keepRunning bool) {
	name, args, ok := parseSlashCommand(input)
	if !ok {
		return false, true
	}

	switch name {
	case "exit", "quit":
		return true, false
	case "help":
		fmt.Println("Commands:")
		fmt.Println("  /help   Show this help")
		fmt.Println("  /tools  List available tools")
		fmt.Println("  /clear  Reset the current chat")
		fmt.Println("  /exit   Leave the chat")
		fmt.Println("  /quit   Alias for /exit")
		fmt.Println()
		fmt.Println("Any other message is sent to the model as plain chat.")
		return true, true
	case "clear":
		a.history = []ollama.Message{{Role: "system", Content: a.systemPrompt}}
		fmt.Println("Chat cleared.")
		return true, true
	case "tools":
		if len(a.skills) == 0 {
			fmt.Println("No tools available.")
			return true, true
		}

		names := make([]string, len(a.skills))
		for i, skill := range a.skills {
			names[i] = skill.Manifest.Name
		}
		fmt.Printf("Tools: %s\n", strings.Join(names, ", "))
		return true, true
	default:
		if args != "" {
			fmt.Printf("Unknown command: /%s %s\n", name, args)
		} else {
			fmt.Printf("Unknown command: /%s\n", name)
		}
		fmt.Println("Type /help for available commands.")
		return true, true
	}
}

// think drives the model round-trips until a plain text response is returned.
// It handles tool calls by executing the matching skill and re-prompting.
// Returns the final reply, the full conversation turn as OpenAI-format messages
// (user message through final assistant reply), and accumulated token counts.
func (a *Agent) think(ctx context.Context, task string) (reply string, messages []training.TrajectoryMessage, inputTokens int, outputTokens int, err error) {
	messages = append(messages, training.TrajectoryMessage{Role: "user", Content: task})

	for {
		printStatus("Thinking...")

		var resp *ollama.ChatResponse
		resp, err = a.client.Chat(ctx, a.history, a.tools)
		clearStatus()

		if err != nil {
			return "", messages, inputTokens, outputTokens, err
		}

		inputTokens += resp.PromptEvalCount
		outputTokens += resp.EvalCount

		msg := resp.Message

		// Extract visible content: prefer the dedicated Thinking field (newer
		// Ollama builds), then fall back to stripping inline <think>…</think>
		// blocks that older builds embed directly in Content.
		visible := stripThinking(msg.Content)

		if a.verbose {
			fmt.Fprintf(os.Stderr, "[debug] role=%q content=%q thinking=%q tool_calls=%d\n",
				msg.Role, visible, msg.Thinking, len(msg.ToolCalls))
			for i, tc := range msg.ToolCalls {
				fmt.Fprintf(os.Stderr, "[debug]   tool_call[%d] name=%q args=%s\n",
					i, tc.Function.Name, tc.Function.Arguments)
			}
		}

		// No tool calls — this is the final text reply.
		if len(msg.ToolCalls) == 0 {
			if visible == "" {
				// The model produced only reasoning tokens and no visible text.
				// Return a fallback so the REPL always prints something useful.
				const fallback = "(no response — the model may still be warming up or the prompt produced only internal reasoning; try rephrasing)"
				messages = append(messages, training.TrajectoryMessage{Role: "assistant", Content: fallback})
				return fallback, messages, inputTokens, outputTokens, nil
			}
			a.history = append(a.history, ollama.Message{
				Role:    "assistant",
				Content: visible,
			})
			messages = append(messages, training.TrajectoryMessage{Role: "assistant", Content: visible})
			return visible, messages, inputTokens, outputTokens, nil
		}

		// Append the assistant's tool-call turn to history.
		// Omit Thinking — it is internal reasoning and wastes context tokens.
		a.history = append(a.history, ollama.Message{
			Role:      "assistant",
			Content:   visible,
			ToolCalls: msg.ToolCalls,
		})

		// Build the assistant trajectory message with tool calls.
		// IDs are generated here so the paired tool-result messages can reference them.
		assistantMsg := training.TrajectoryMessage{
			Role:             "assistant",
			Content:          visible,
			ReasoningContent: msg.Thinking,
			ToolCalls:        make([]training.TrajectoryToolCall, len(msg.ToolCalls)),
		}
		for i, tc := range msg.ToolCalls {
			assistantMsg.ToolCalls[i] = training.TrajectoryToolCall{
				ID:   newToolCallID(),
				Type: "function",
				Function: training.TrajectoryToolFunction{
					Name:      tc.Function.Name,
					Arguments: string(tc.Function.Arguments),
				},
			}
		}
		messages = append(messages, assistantMsg)

		// Execute each requested tool, append results to history and messages.
		for i, tc := range msg.ToolCalls {
			name := tc.Function.Name
			callID := assistantMsg.ToolCalls[i].ID
			printStatus(fmt.Sprintf("Running tool: %s...", name))

			skill, found := a.findSkill(name)
			var result string
			if !found {
				result = fmt.Sprintf("Error: tool %q is not available", name)
			} else if pathErr := a.checkSkillPaths(skill, tc.Function.Arguments); pathErr != nil {
				result = pathErr.Error()
			} else {
				var toolErr error
				result, toolErr = skills.Execute(ctx, skill, tc.Function.Arguments, a.allowedPaths)
				if toolErr != nil {
					result = fmt.Sprintf("Error executing tool %q: %v", name, toolErr)
				}
			}
			clearStatus()

			a.history = append(a.history, ollama.Message{
				Role:    "tool",
				Name:    name,
				Content: result,
			})
			messages = append(messages, training.TrajectoryMessage{
				Role:       "tool",
				ToolCallID: callID,
				Name:       name,
				Content:    result,
			})
		}

		// Loop back to get the synthesis response.
	}
}

// newToolCallID returns a short random ID for linking tool calls to their results.
func newToolCallID() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return "call_" + hex.EncodeToString(b)
}

// checkSkillPaths validates each path-type argument declared in the skill's
// manifest against the agent's allowedPaths list. It returns an error suitable
// for returning to the LLM as a tool result if any path is denied.
func (a *Agent) checkSkillPaths(skill skills.Skill, argsJSON json.RawMessage) error {
	if len(a.allowedPaths) == 0 || len(skill.Manifest.PathParams) == 0 {
		return nil
	}

	var params map[string]json.RawMessage
	if err := json.Unmarshal(argsJSON, &params); err != nil {
		return nil // malformed JSON — let Execute handle it
	}

	for _, paramName := range skill.Manifest.PathParams {
		raw, ok := params[paramName]
		if !ok {
			continue
		}
		var pathVal string
		if err := json.Unmarshal(raw, &pathVal); err != nil || pathVal == "" {
			continue
		}
		if err := checkPathAllowed(pathVal, a.allowedPaths); err != nil {
			return err
		}
	}
	return nil
}

// findSkill returns the Skill whose manifest name matches, or false.
func (a *Agent) findSkill(name string) (skills.Skill, bool) {
	for _, s := range a.skills {
		if s.Manifest.Name == name {
			return s, true
		}
	}
	return skills.Skill{}, false
}

// printStatus writes a status message to stderr that can be overwritten.
func printStatus(msg string) {
	fmt.Fprint(os.Stderr, msg)
}

// clearStatus overwrites the current status line on stderr.
func clearStatus() {
	fmt.Fprint(os.Stderr, "\r                                                            \r")
}
