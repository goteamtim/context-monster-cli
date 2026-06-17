package training

import (
	"encoding/json"
	"time"
)

// RecordedToolCall captures a model-invoked tool call for training records.
// It is intentionally decoupled from the ollama API types so the training
// format is stable across API changes.
type RecordedToolCall struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// Step captures one tool invocation and its result within an episode.
// Observation is what the model "saw" that led to this tool call: the original
// task for the first step, or the prior tool result for subsequent steps.
// Reasoning captures the model's chain-of-thought when available (reasoning
// models only — omitted when empty).
type Step struct {
	Observation string           `json:"observation"`
	Reasoning   string           `json:"reasoning,omitempty"`
	ToolCall    RecordedToolCall `json:"tool_call"`
	ToolResult  string           `json:"tool_result"`
	Timestamp   time.Time        `json:"timestamp"`
}

// EpisodeMetadata captures model and runtime context needed for reproducibility
// and eval harness comparisons.
type EpisodeMetadata struct {
	Model         string  `json:"model"`
	Provider      string  `json:"provider"`
	PersonaName   string  `json:"persona_name,omitempty"`
	Temperature   float64 `json:"temperature,omitempty"`
	TopP          float64 `json:"top_p,omitempty"`
	ContextWindow int     `json:"context_window,omitempty"`
	InputTokens   int     `json:"input_tokens"`
	OutputTokens  int     `json:"output_tokens"`
	TotalTokens   int     `json:"total_tokens"`
}

// Episode is the atomic unit of training data: one user task, the steps the
// model took via tool calls, and the final answer it produced. One episode
// corresponds to one call to think() — a single user turn through to the
// model's final response.
//
// Success and SuccessReason default to their zero values and are intended for
// manual or LLM-assisted annotation after the fact.
type Episode struct {
	ID            string          `json:"id"`
	Task          string          `json:"task"`
	Steps         []Step          `json:"steps"`
	FinalAnswer   string          `json:"final_answer"`
	Success       bool            `json:"success"`
	SuccessReason string          `json:"success_reason,omitempty"`
	Metadata      EpisodeMetadata `json:"metadata"`
	StartedAt     time.Time       `json:"started_at"`
	CompletedAt   time.Time       `json:"completed_at"`
}
