package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// ToolFunctionParam describes a single parameter in a tool's schema.
type ToolFunctionParam struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

// ToolFunctionParameters is the JSON Schema-style parameters block.
type ToolFunctionParameters struct {
	Type       string                       `json:"type"`
	Properties map[string]ToolFunctionParam `json:"properties"`
	Required   []string                     `json:"required"`
}

// ToolFunction is the function definition inside a Tool.
type ToolFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  ToolFunctionParameters `json:"parameters"`
}

// Tool represents a callable tool declared in a chat request.
type Tool struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

// ToolCallFunction holds the name and arguments of a model-invoked tool call.
type ToolCallFunction struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// ToolCall represents a single tool invocation returned by the model.
type ToolCall struct {
	Function ToolCallFunction `json:"function"`
}

// Message is a single entry in the conversation history.
// ToolCalls is populated only on assistant messages that invoke tools.
// Name identifies the tool that produced a result; required on tool-role messages.
type Message struct {
	Role      string     `json:"role"`
	Name      string     `json:"name,omitempty"`
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// ChatRequest is the payload sent to /api/chat.
type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Tools    []Tool    `json:"tools,omitempty"`
	Stream   bool      `json:"stream"`
	Options  *Options  `json:"options,omitempty"`
}

// ChatResponse is the response body from /api/chat.
type ChatResponse struct {
	Message Message `json:"message"`
}

// Options specifies optional model-level parameters sent with each chat request.
type Options struct {
	NumCtx     int `json:"num_ctx,omitempty"`
	NumPredict int `json:"num_predict,omitempty"`
}

// Client is an HTTP client for the Ollama chat API.
type Client struct {
	baseURL    string
	model      string
	httpClient *http.Client
	options    *Options
}

// New creates a new Client targeting baseURL with the given model.
// opts may be empty to use Ollama's defaults.
func New(baseURL, model string, opts *Options) *Client {
	return &Client{
		baseURL:    baseURL,
		model:      model,
		httpClient: &http.Client{},
		options:    opts,
	}
}

// Chat sends the conversation history and available tools to Ollama and returns
// the model's response. It uses the non-streaming endpoint.
func (c *Client) Chat(ctx context.Context, messages []Message, tools []Tool) (*ChatResponse, error) {
	reqBody := ChatRequest{
		Model:    c.model,
		Messages: messages,
		Tools:    tools,
		Stream:   false,
		Options:  c.options,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal chat request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to build HTTP request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("could not reach Ollama at %s — is it running? (%w)", c.baseURL, err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		// handled below
	case http.StatusNotFound:
		return nil, fmt.Errorf("model %q not found — run: ollama pull %s", c.model, c.model)
	default:
		return nil, fmt.Errorf("Ollama returned unexpected status %d", resp.StatusCode)
	}

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("failed to decode Ollama response: %w", err)
	}

	return &chatResp, nil
}
