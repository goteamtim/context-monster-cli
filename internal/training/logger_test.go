package training

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// minimalTrajectory returns a well-formed Trajectory for use in tests.
func minimalTrajectory(task string) Trajectory {
	return Trajectory{
		Messages: []TrajectoryMessage{
			{Role: "user", Content: task},
			{Role: "assistant", Content: "done"},
		},
		Metadata: TrajectoryMetadata{
			ID:          "test-id",
			Model:       "qwen3.5:4b",
			Provider:    "ollama",
			InputTokens: 10,
			OutputTokens: 5,
			TotalTokens: 15,
			StartedAt:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			CompletedAt: time.Date(2026, 1, 1, 0, 0, 1, 0, time.UTC),
		},
	}
}

// readLines returns all non-empty lines from the file at path.
func readLines(t *testing.T, path string) []string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %q: %v", path, err)
	}
	defer f.Close()

	var lines []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if line := sc.Text(); line != "" {
			lines = append(lines, line)
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("scan %q: %v", path, err)
	}
	return lines
}

func TestNew_CreatesOutputDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "personas", "my_agent", "training")
	if _, err := New(dir); err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("output dir not created: %v", err)
	}
}

func TestNew_OutputFileIsTrajectoriesdotJSONL(t *testing.T) {
	dir := t.TempDir()
	logger, err := New(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	want := filepath.Join(dir, "trajectories.jsonl")
	if logger.path != want {
		t.Fatalf("logger.path = %q, want %q", logger.path, want)
	}
}

func TestAppend_WritesValidJSONL(t *testing.T) {
	dir := t.TempDir()
	logger, _ := New(dir)

	if err := logger.Append(minimalTrajectory("list all Go files")); err != nil {
		t.Fatalf("Append: %v", err)
	}

	lines := readLines(t, logger.path)
	if len(lines) != 1 {
		t.Fatalf("got %d lines, want 1", len(lines))
	}

	var got Trajectory
	if err := json.Unmarshal([]byte(lines[0]), &got); err != nil {
		t.Fatalf("line is not valid JSON: %v", err)
	}
	if len(got.Messages) != 2 || got.Messages[0].Content != "list all Go files" {
		t.Fatalf("unexpected trajectory content: %+v", got)
	}
}

func TestAppend_AppendsMultipleTrajectories(t *testing.T) {
	dir := t.TempDir()
	logger, _ := New(dir)

	const n = 3
	for i := 0; i < n; i++ {
		traj := minimalTrajectory(fmt.Sprintf("task %d", i))
		if err := logger.Append(traj); err != nil {
			t.Fatalf("Append %d: %v", i, err)
		}
	}

	lines := readLines(t, logger.path)
	if len(lines) != n {
		t.Fatalf("got %d lines, want %d", len(lines), n)
	}

	// Verify each line is valid JSON and round-trips the task field.
	for i, line := range lines {
		var got Trajectory
		if err := json.Unmarshal([]byte(line), &got); err != nil {
			t.Fatalf("line %d is not valid JSON: %v", i, err)
		}
		want := fmt.Sprintf("task %d", i)
		if len(got.Messages) == 0 || got.Messages[0].Content != want {
			t.Fatalf("line %d: got task %q, want %q", i, got.Messages[0].Content, want)
		}
	}
}

func TestAppend_PreservesToolCallStructure(t *testing.T) {
	dir := t.TempDir()
	logger, _ := New(dir)

	traj := Trajectory{
		Messages: []TrajectoryMessage{
			{Role: "user", Content: "find all .go files"},
			{
				Role: "assistant",
				ToolCalls: []TrajectoryToolCall{
					{
						ID:   "call_abc123",
						Type: "function",
						Function: TrajectoryToolFunction{
							Name:      "file_search",
							Arguments: `{"dir":".","ext":".go"}`,
						},
					},
				},
			},
			{
				Role:       "tool",
				ToolCallID: "call_abc123",
				Name:       "file_search",
				Content:    "./main.go\n./engine.go",
			},
			{Role: "assistant", Content: "Found 2 Go files."},
		},
		Metadata: TrajectoryMetadata{ID: "tool-test"},
	}

	if err := logger.Append(traj); err != nil {
		t.Fatalf("Append: %v", err)
	}

	lines := readLines(t, logger.path)
	var got Trajectory
	if err := json.Unmarshal([]byte(lines[0]), &got); err != nil {
		t.Fatalf("not valid JSON: %v", err)
	}

	assistantMsg := got.Messages[1]
	if len(assistantMsg.ToolCalls) != 1 {
		t.Fatalf("got %d tool_calls, want 1", len(assistantMsg.ToolCalls))
	}
	if assistantMsg.ToolCalls[0].ID != "call_abc123" {
		t.Fatalf("tool_call id = %q, want %q", assistantMsg.ToolCalls[0].ID, "call_abc123")
	}

	toolMsg := got.Messages[2]
	if toolMsg.ToolCallID != "call_abc123" {
		t.Fatalf("tool message tool_call_id = %q, want %q", toolMsg.ToolCallID, "call_abc123")
	}
	if toolMsg.Name != "file_search" {
		t.Fatalf("tool message name = %q, want %q", toolMsg.Name, "file_search")
	}
}

func TestAppend_PreservesReasoningContent(t *testing.T) {
	dir := t.TempDir()
	logger, _ := New(dir)

	traj := Trajectory{
		Messages: []TrajectoryMessage{
			{Role: "user", Content: "summarise recent changes"},
			{
				Role:             "assistant",
				ReasoningContent: "I should check the git log before answering.",
				Content:          "Here is a summary...",
			},
		},
		Metadata: TrajectoryMetadata{ID: "reasoning-test"},
	}

	if err := logger.Append(traj); err != nil {
		t.Fatalf("Append: %v", err)
	}

	lines := readLines(t, logger.path)
	var got Trajectory
	if err := json.Unmarshal([]byte(lines[0]), &got); err != nil {
		t.Fatalf("not valid JSON: %v", err)
	}

	msg := got.Messages[1]
	if msg.ReasoningContent != "I should check the git log before answering." {
		t.Fatalf("reasoning_content = %q, want reasoning preserved", msg.ReasoningContent)
	}
}
