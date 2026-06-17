package training

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Logger writes episodes as newline-delimited JSON (JSONL) to a single file.
// Each call to Append writes exactly one JSON object followed by a newline.
type Logger struct {
	path string
}

// New creates a Logger that appends to <outputDir>/episodes.jsonl.
// The directory is created if it does not already exist.
func New(outputDir string) (*Logger, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return nil, fmt.Errorf("training: could not create output dir %q: %w", outputDir, err)
	}
	return &Logger{path: filepath.Join(outputDir, "episodes.jsonl")}, nil
}

// NewEpisodeID returns a random 32-character hex string for use as Episode.ID.
func NewEpisodeID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// Append marshals ep as a single JSON line and appends it to the log file.
// The file is opened and closed on every call so partial writes from
// concurrent or interrupted runs do not corrupt prior entries.
func (l *Logger) Append(ep Episode) error {
	line, err := json.Marshal(ep)
	if err != nil {
		return fmt.Errorf("training: could not marshal episode: %w", err)
	}

	f, err := os.OpenFile(l.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("training: could not open log file %q: %w", l.path, err)
	}
	defer f.Close()

	_, err = fmt.Fprintf(f, "%s\n", line)
	return err
}
