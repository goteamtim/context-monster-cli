package agent

import "strings"

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
