package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/goteamtim/context-monster-cli/internal/ollama"
	"github.com/goteamtim/context-monster-cli/internal/training"
)

// stubClient implements chatClient for testing without a live Ollama process.
// Responses are consumed in order; an error at index i is returned for call i.
type stubClient struct {
	responses []*ollama.ChatResponse
	errors    []error
	calls     int
}

func (s *stubClient) Chat(ctx context.Context, _ []ollama.Message, _ []ollama.Tool) (*ollama.ChatResponse, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	i := s.calls
	s.calls++
	if i < len(s.errors) && s.errors[i] != nil {
		return nil, s.errors[i]
	}
	if i < len(s.responses) {
		return s.responses[i], nil
	}
	return nil, fmt.Errorf("stub: no response configured for call %d", i)
}

func newTestAgent(client chatClient) *Agent {
	const prompt = "system"
	return &Agent{
		client:       client,
		systemPrompt: prompt,
		history:      []ollama.Message{{Role: "system", Content: prompt}},
		meta:         training.TrajectoryMetadata{},
	}
}

func assistantResp(content string) *ollama.ChatResponse {
	return &ollama.ChatResponse{Message: ollama.Message{Role: "assistant", Content: content}}
}

func TestThink_plainTextReply(t *testing.T) {
	stub := &stubClient{responses: []*ollama.ChatResponse{assistantResp("hello world")}}
	reply, _, _, _, err := newTestAgent(stub).think(context.Background(), "hi")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reply != "hello world" {
		t.Fatalf("got %q, want %q", reply, "hello world")
	}
}

func TestThink_emptyContentReturnsFallback(t *testing.T) {
	stub := &stubClient{responses: []*ollama.ChatResponse{assistantResp("")}}
	reply, _, _, _, err := newTestAgent(stub).think(context.Background(), "hi")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reply == "" {
		t.Fatal("expected non-empty fallback reply for empty model response")
	}
}

func TestThink_cancelledContextReturnsError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, _, _, _, err := newTestAgent(&stubClient{}).think(ctx, "hi")
	if err == nil {
		t.Fatal("expected error for cancelled context, got nil")
	}
}

// TestThink_unknownToolCallCyclesBackForSynthesis verifies that when the model
// requests a tool that doesn't exist, the agent feeds the error back and
// makes a second Chat call to get a synthesis response.
func TestThink_unknownToolCallCyclesBackForSynthesis(t *testing.T) {
	stub := &stubClient{
		responses: []*ollama.ChatResponse{
			// First call: model requests an unknown tool.
			{Message: ollama.Message{
				Role: "assistant",
				ToolCalls: []ollama.ToolCall{
					{Function: ollama.ToolCallFunction{
						Name:      "nonexistent_skill",
						Arguments: json.RawMessage(`{}`),
					}},
				},
			}},
			// Second call: model synthesises after receiving the "not available" error.
			assistantResp("I cannot do that."),
		},
	}

	reply, _, _, _, err := newTestAgent(stub).think(context.Background(), "do something")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reply != "I cannot do that." {
		t.Fatalf("got %q, want %q", reply, "I cannot do that.")
	}
	if stub.calls != 2 {
		t.Fatalf("expected 2 Chat calls, got %d", stub.calls)
	}
}
