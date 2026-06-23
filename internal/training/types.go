package training

import (
	"time"
)

// TrajectoryMessage is a single message in the conversation using the OpenAI
// messages format, so the trajectory file is directly consumable by SFT pipelines
// (Axolotl, LLaMA-Factory, unsloth) without any conversion step.
//
// ReasoningContent is not part of the OpenAI spec and is ignored by training
// frameworks, but preserved for trajectory analysis. The name matches the field
// used by DeepSeek and Qwen reasoning models so parsers find a familiar name.
//
// ToolCallID and Name are only populated on role "tool" messages. ToolCallID
// links a result back to the assistant tool_call that requested it; Name
// identifies which function produced the result (required for full OpenAI spec
// compliance — some pipelines use it, some ignore it).
type TrajectoryMessage struct {
	Role             string               `json:"role"`
	Content          string               `json:"content,omitempty"`
	ReasoningContent string               `json:"reasoning_content,omitempty"`
	ToolCalls        []TrajectoryToolCall `json:"tool_calls,omitempty"`
	ToolCallID       string               `json:"tool_call_id,omitempty"`
	Name             string               `json:"name,omitempty"`
}

// TrajectoryToolCall is an OpenAI-format tool call on an assistant message.
type TrajectoryToolCall struct {
	ID       string                `json:"id"`
	Type     string                `json:"type"` // always "function"
	Function TrajectoryToolFunction `json:"function"`
}

// TrajectoryToolFunction holds the name and JSON-encoded arguments of a tool call.
// Arguments is a JSON string (not an object) to match the OpenAI wire format.
type TrajectoryToolFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// TrajectoryMetadata captures model and runtime context for trajectory analysis
// and eval. Fine-tuning pipelines ignore this field.
//
// Success and SuccessReason default to their zero values and are intended for
// manual or LLM-assisted annotation after the fact.
type TrajectoryMetadata struct {
	ID            string    `json:"id"`
	Model         string    `json:"model"`
	Provider      string    `json:"provider"`
	PersonaName   string    `json:"persona_name,omitempty"`
	ContextWindow int       `json:"context_window,omitempty"`
	InputTokens   int       `json:"input_tokens"`
	OutputTokens  int       `json:"output_tokens"`
	TotalTokens   int       `json:"total_tokens"`
	Success       bool      `json:"success"`
	SuccessReason string    `json:"success_reason,omitempty"`
	StartedAt     time.Time `json:"started_at"`
	CompletedAt   time.Time `json:"completed_at"`
}

// Trajectory is the atomic unit of recorded output: one user task and the full
// conversation as OpenAI-format messages, paired with analysis metadata. One
// trajectory corresponds to one call to think() — a single user turn through
// to the model's final response.
//
// The messages array is directly consumable by SFT pipelines. The metadata field
// is ignored by training frameworks but used by analysis and eval tooling.
type Trajectory struct {
	Messages []TrajectoryMessage `json:"messages"`
	Metadata TrajectoryMetadata  `json:"metadata"`
}
