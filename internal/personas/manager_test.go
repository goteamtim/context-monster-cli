package personas

import (
	"strings"
	"testing"
)

func TestLoad_missingDirectoryIncludesPathInError(t *testing.T) {
	_, err := Load("/nonexistent/personas/dir")
	if err == nil {
		t.Fatal("expected error for missing directory, got nil")
	}
	if !strings.Contains(err.Error(), "/nonexistent/personas/dir") {
		t.Fatalf("error %q does not mention the directory path", err.Error())
	}
}
