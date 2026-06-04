package agent

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"context-monster-cli/pkg/ollama"
	"context-monster-cli/pkg/skills"
)

// Agent orchestrates the REPL loop and multi-turn tool-calling conversation.
type Agent struct {
	client       *ollama.Client
	skills       []skills.Skill
	systemPrompt string
	history      []ollama.Message
	tools        []ollama.Tool
	verbose      bool
}

// New creates an Agent wired with the given Ollama client and loaded skills.
// systemPrompt is injected as the first message in the conversation history.
func New(client *ollama.Client, loadedSkills []skills.Skill, systemPrompt string, verbose bool) *Agent {
	tools := make([]ollama.Tool, len(loadedSkills))
	for i, s := range loadedSkills {
		tools[i] = s.ToOllamaTool()
	}

	return &Agent{
		client:       client,
		skills:       loadedSkills,
		systemPrompt: systemPrompt,
		tools:        tools,
		verbose:      verbose,
		history: []ollama.Message{
			{Role: "system", Content: systemPrompt},
		},
	}
}

// Run starts the interactive REPL. It reads lines from stdin, drives the
// multi-turn conversation, and writes assistant replies to stdout.
func (a *Agent) Run() {
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

		reply, err := a.think(context.Background())
		if err != nil {
			fmt.Fprintf(os.Stderr, "\nError: %v\n\n", err)
			// Remove the user message so history stays consistent on error
			a.history = a.history[:len(a.history)-1]
			continue
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
func (a *Agent) think(ctx context.Context) (string, error) {
	for {
		printStatus("Thinking...")

		resp, err := a.client.Chat(ctx, a.history, a.tools)
		clearStatus()

		if err != nil {
			return "", err
		}

		msg := resp.Message

		if a.verbose {
			fmt.Fprintf(os.Stderr, "[debug] role=%q content=%q tool_calls=%d\n",
				msg.Role, msg.Content, len(msg.ToolCalls))
			for i, tc := range msg.ToolCalls {
				fmt.Fprintf(os.Stderr, "[debug]   tool_call[%d] name=%q args=%s\n",
					i, tc.Function.Name, tc.Function.Arguments)
			}
		}

		// No tool calls — this is the final text reply.
		if len(msg.ToolCalls) == 0 {
			a.history = append(a.history, ollama.Message{
				Role:    "assistant",
				Content: msg.Content,
			})
			return msg.Content, nil
		}

		// Append the assistant's tool-call turn to history.
		a.history = append(a.history, ollama.Message{
			Role:      "assistant",
			Content:   msg.Content,
			ToolCalls: msg.ToolCalls,
		})

		// Execute each requested tool and append results to history.
		for _, tc := range msg.ToolCalls {
			name := tc.Function.Name
			printStatus(fmt.Sprintf("Running tool: %s...", name))

			skill, found := a.findSkill(name)
			var result string
			if !found {
				result = fmt.Sprintf("Error: tool %q is not available", name)
			} else {
				var toolErr error
				result, toolErr = skills.Execute(skill, tc.Function.Arguments)
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
		}

		// Loop back to get the synthesis response.
	}
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
