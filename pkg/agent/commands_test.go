package agent

import (
	"bytes"
	"io"
	"os"
	"testing"

	"context-monster-cli/pkg/ollama"
)

func TestParseSlashCommand(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantName string
		wantArgs string
		wantOK   bool
	}{
		{name: "plain text", input: "hello", wantOK: false},
		{name: "exit", input: "/exit", wantName: "exit", wantOK: true},
		{name: "quit alias", input: "/quit", wantName: "quit", wantOK: true},
		{name: "help with spaces", input: " /help ", wantName: "help", wantOK: true},
		{name: "command with args", input: "/tools one two", wantName: "tools", wantArgs: "one two", wantOK: true},
		{name: "bare slash", input: "/", wantOK: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotArgs, gotOK := parseSlashCommand(tt.input)
			if gotName != tt.wantName || gotArgs != tt.wantArgs || gotOK != tt.wantOK {
				t.Fatalf("parseSlashCommand(%q) = (%q, %q, %v), want (%q, %q, %v)", tt.input, gotName, gotArgs, gotOK, tt.wantName, tt.wantArgs, tt.wantOK)
			}
		})
	}
}

func TestHandleSlashCommand(t *testing.T) {
	agent := &Agent{systemPrompt: "system", history: []ollama.Message{{Role: "system", Content: "system"}}}

	output := captureStdout(t, func() {
		handled, keepRunning := agent.handleSlashCommand("/clear")
		if !handled || !keepRunning {
			t.Fatalf("/clear returned handled=%v keepRunning=%v", handled, keepRunning)
		}
	})
	if output != "Chat cleared.\n" {
		t.Fatalf("/clear output = %q, want %q", output, "Chat cleared.\n")
	}
	if len(agent.history) != 1 || agent.history[0].Role != "system" || agent.history[0].Content != "system" {
		t.Fatalf("/clear did not reset history: %#v", agent.history)
	}

	output = captureStdout(t, func() {
		handled, keepRunning := agent.handleSlashCommand("/exit")
		if !handled || keepRunning {
			t.Fatalf("/exit returned handled=%v keepRunning=%v", handled, keepRunning)
		}
	})
	if output != "" {
		t.Fatalf("/exit output = %q, want empty", output)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w

	outputCh := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		outputCh <- buf.String()
	}()

	fn()
	_ = w.Close()
	os.Stdout = oldStdout

	return <-outputCh
}

func TestStripThinking(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "no thinking block", input: "Hello!", want: "Hello!"},
		{name: "only thinking block", input: "<think>reasoning</think>", want: ""},
		{name: "thinking then reply", input: "<think>\nsome reasoning\n</think>\nHello!", want: "Hello!"},
		{name: "multiple blocks", input: "<think>a</think> middle <think>b</think> end", want: "middle  end"},
		{name: "empty string", input: "", want: ""},
		{name: "whitespace after strip", input: "<think>x</think>   \n   ", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripThinking(tt.input)
			if got != tt.want {
				t.Fatalf("stripThinking(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
