package skills

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const skillTimeout = 30 * time.Second

// Load scans dir for subdirectories that contain a manifest.json and returns
// the parsed skills. Directories without a manifest are silently skipped.
func Load(dir string) ([]Skill, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("cannot read skills directory %q: %w", dir, err)
	}

	var skills []Skill
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		manifestPath := filepath.Join(dir, entry.Name(), "manifest.json")
		data, err := os.ReadFile(manifestPath)
		if err != nil {
			continue // not a skill directory — skip silently
		}

		var m Manifest
		if err := json.Unmarshal(data, &m); err != nil {
			return nil, fmt.Errorf("invalid manifest at %s: %w", manifestPath, err)
		}

		agentsPath := filepath.Join(dir, entry.Name(), "AGENTS.md")
		if agentsContent, err := os.ReadFile(agentsPath); err == nil {
			m.Description = string(agentsContent)
		}

		skills = append(skills, Skill{
			Manifest: m,
			Dir:      filepath.Join(dir, entry.Name()),
		})
	}

	return skills, nil
}

// FindByName returns the first Skill whose manifest name matches, plus a found bool.
func FindByName(skillList []Skill, name string) (Skill, bool) {
	for _, s := range skillList {
		if s.Manifest.Name == name {
			return s, true
		}
	}
	return Skill{}, false
}

// Execute runs a skill's command with the provided JSON arguments and returns
// the trimmed stdout. A context deadline of skillTimeout is applied automatically.
// allowedPaths is forwarded as CM_ALLOWED_PATHS so shell-script skills can
// enforce the same restrictions; pass nil for unrestricted access.
func Execute(skill Skill, argsJSON json.RawMessage, allowedPaths []string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), skillTimeout)
	defer cancel()

	parts := strings.Fields(skill.Manifest.Command)
	if len(parts) == 0 {
		return "", fmt.Errorf("skill %q has an empty command", skill.Manifest.Name)
	}

	// Resolve the binary path relative to the skill directory when it starts with "."
	binary := parts[0]
	if strings.HasPrefix(binary, ".") {
		binary = filepath.Join(skill.Dir, binary)
	}

	// Append the JSON args string as the final positional argument
	cmdArgs := make([]string, len(parts)-1, len(parts))
	copy(cmdArgs, parts[1:])
	cmdArgs = append(cmdArgs, string(argsJSON))

	cmd := exec.CommandContext(ctx, binary, cmdArgs...)
	// Do not set cmd.Dir — skills inherit the agent's working directory so that
	// relative paths in tool arguments resolve correctly from the user's CWD.
	// Inject the skills root so meta-skills like build_skill can locate it even
	// under go run, where os.Executable() returns a temp path.
	cmd.Env = append(os.Environ(), fmt.Sprintf("CM_SKILLS_DIR=%s", filepath.Dir(skill.Dir)))
	if len(allowedPaths) > 0 {
		cmd.Env = append(cmd.Env, fmt.Sprintf("CM_ALLOWED_PATHS=%s", strings.Join(allowedPaths, ",")))
	}

	out, err := cmd.Output()
	if err != nil {
		if ctx.Err() != nil {
			return "", fmt.Errorf("skill %q timed out after %s", skill.Manifest.Name, skillTimeout)
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("skill %q failed: %s", skill.Manifest.Name, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return "", fmt.Errorf("skill %q failed: %w", skill.Manifest.Name, err)
	}

	return strings.TrimSpace(string(out)), nil
}
