package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// checkPathAllowed reports whether rawPath is permitted by allowedPatterns.
//
// Rules (applied in order, first match wins):
//   - If allowedPatterns is empty, all paths are allowed (backward-compatible).
//   - Each pattern is resolved to an absolute path (relative patterns are
//     resolved from the current working directory at call time).
//   - A pattern ending in "/**" or "/" is treated as a directory prefix: any
//     file whose absolute path starts with <pattern>/ is allowed.
//   - A pattern containing '*' or '?' is matched via filepath.Match.
//   - Otherwise the pattern allows an exact file match OR any path whose
//     absolute path has the pattern as a directory prefix (i.e. the pattern
//     names a directory, and rawPath is inside it).
func checkPathAllowed(rawPath string, allowedPatterns []string) error {
	if len(allowedPatterns) == 0 {
		return nil
	}

	absTarget, err := filepath.Abs(rawPath)
	if err != nil {
		return fmt.Errorf("resolving path %q: %w", rawPath, err)
	}
	// Normalise separators for consistent comparison.
	absTarget = filepath.Clean(absTarget)

	cwd, _ := os.Getwd()

	for _, pattern := range allowedPatterns {
		// Resolve relative patterns against cwd.
		absPattern := pattern
		if !filepath.IsAbs(pattern) {
			absPattern = filepath.Join(cwd, pattern)
		}
		absPattern = filepath.Clean(absPattern)

		// Strip a trailing "/**" suffix — means "this dir and everything below".
		if strings.HasSuffix(absPattern, string(os.PathSeparator)+"**") {
			absPattern = strings.TrimSuffix(absPattern, string(os.PathSeparator)+"**")
		}

		// Glob pattern: delegate to filepath.Match.
		if strings.ContainsAny(absPattern, "*?") {
			matched, err := filepath.Match(absPattern, absTarget)
			if err == nil && matched {
				return nil
			}
			continue
		}

		// Exact file match.
		if absTarget == absPattern {
			return nil
		}

		// Directory prefix match: absTarget is inside absPattern directory.
		prefix := absPattern + string(os.PathSeparator)
		if strings.HasPrefix(absTarget, prefix) {
			return nil
		}
	}

	return fmt.Errorf("access denied: %q is outside the persona's allowed paths", rawPath)
}
