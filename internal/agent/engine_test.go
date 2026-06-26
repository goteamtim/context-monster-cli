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

// makeHistory returns a history slice with a pinned system message followed by
// n additional user messages, simulating a long conversation.
func makeHistory(n int) []ollama.Message {
	msgs := make([]ollama.Message, n+1)
	msgs[0] = ollama.Message{Role: "system", Content: "system prompt"}
	for i := 1; i <= n; i++ {
		msgs[i] = ollama.Message{Role: "user", Content: fmt.Sprintf("message %d", i)}
	}
	return msgs
}

func TestMaybeCompact_noopWithoutContextWindow(t *testing.T) {
	a := newTestAgent(nil)
	a.meta.ContextWindow = 0
	a.history = makeHistory(10)
	a.maybeCompact(9000)
	if len(a.history) != 11 {
		t.Fatalf("expected history unchanged (11 messages), got %d", len(a.history))
	}
}

func TestMaybeCompact_noopUnderThreshold(t *testing.T) {
	a := newTestAgent(nil)
	a.meta.ContextWindow = 1000
	a.history = makeHistory(10)
	a.maybeCompact(750) // 75% — under the 80% threshold
	if len(a.history) != 11 {
		t.Fatalf("expected no compaction at 75%% usage, got %d messages", len(a.history))
	}
}

func TestMaybeCompact_dropsOldestMessages(t *testing.T) {
	a := newTestAgent(nil)
	a.meta.ContextWindow = 1000
	a.history = makeHistory(10)
	a.maybeCompact(900) // 90% — over threshold
	if len(a.history) >= 11 {
		t.Fatalf("expected messages to be dropped, history still has %d messages", len(a.history))
	}
	if a.history[0].Role != "system" {
		t.Fatalf("system message was not kept at index 0, got role %q", a.history[0].Role)
	}
}

func TestMaybeCompact_preservesSystemAndMinimum(t *testing.T) {
	a := newTestAgent(nil)
	a.meta.ContextWindow = 100
	// Only 2 messages total — should not drop anything (would leave just system).
	a.history = makeHistory(1) // system + 1 message
	a.maybeCompact(100)
	if len(a.history) != 2 {
		t.Fatalf("expected history unchanged at minimum size, got %d messages", len(a.history))
	}
}

func TestMaybeCompact_heavyUsageDropsEnoughToReachTarget(t *testing.T) {
	a := newTestAgent(nil)
	a.meta.ContextWindow = 1000
	a.history = makeHistory(20) // system + 20 messages
	a.maybeCompact(950)         // 95% used — should drop to ~70%

	// After compaction, kept messages × avgTokensPerMessage should be ≤ target.
	// We can't verify tokens directly, but we can check a meaningful number were dropped.
	if len(a.history) >= 21 {
		t.Fatalf("expected significant compaction, got %d messages (no drop)", len(a.history))
	}
	if a.history[0].Role != "system" {
		t.Fatalf("system message must remain at index 0")
	}
}

// TestThink_triggersCompactionOnHighUsage verifies that think() calls
// maybeCompact after a response whose PromptEvalCount exceeds the threshold,
// so the history is shorter after the call than it would be without compaction.
func TestThink_triggersCompactionOnHighUsage(t *testing.T) {
	stub := &stubClient{
		responses: []*ollama.ChatResponse{
			{
				Message:         ollama.Message{Role: "assistant", Content: "reply"},
				PromptEvalCount: 900, // 90% of contextWindow=1000 → triggers compaction
			},
		},
	}
	a := newTestAgent(stub)
	a.meta.ContextWindow = 1000
	// Seed history with enough turns to give compaction something to drop.
	a.history = makeHistory(10) // system + 10 messages = 11 total

	_, _, _, _, err := a.think(context.Background(), "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Without compaction: 11 (existing) + 1 (assistant reply from think) = 12.
	// With compaction some messages are dropped, so total must be < 12.
	if len(a.history) >= 12 {
		t.Fatalf("expected compaction to reduce history, got %d messages", len(a.history))
	}
	if a.history[0].Role != "system" {
		t.Fatalf("system message must remain pinned at index 0")
	}
}
