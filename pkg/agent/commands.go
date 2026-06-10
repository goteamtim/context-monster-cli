package agent

import (
	"regexp"
	"strings"
)

// thinkRe matches one or more <think>…</think> blocks, including newlines,
// as produced by qwen3 reasoning models when Ollama embeds thinking inline.
var thinkRe = regexp.MustCompile(`(?s)<think>.*?</think>`)

// stripThinking removes embedded chain-of-thought blocks and returns the
// visible response text. Returns an empty string if nothing remains.
func stripThinking(content string) string {
	stripped := thinkRe.ReplaceAllString(content, "")
	return strings.TrimSpace(stripped)
}

// parseSlashCommand splits a leading slash command into its name and args.
// It returns ok=false for non-command input.
func parseSlashCommand(input string) (name string, args string, ok bool) {
	input = strings.TrimSpace(input)
	if !strings.HasPrefix(input, "/") {
		return "", "", false
	}

	trimmed := strings.TrimSpace(strings.TrimPrefix(input, "/"))
	if trimmed == "" {
		return "", "", true
	}

	parts := strings.Fields(trimmed)
	name = strings.ToLower(parts[0])
	if len(parts) > 1 {
		args = strings.Join(parts[1:], " ")
	}
	return name, args, true
}
