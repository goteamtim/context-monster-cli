package agent

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckPathAllowed_EmptyPatterns(t *testing.T) {
	// Empty allow-list must permit any path.
	if err := checkPathAllowed("/some/random/file.go", nil); err != nil {
		t.Fatalf("expected nil for empty patterns, got %v", err)
	}
	if err := checkPathAllowed("/some/random/file.go", []string{}); err != nil {
		t.Fatalf("expected nil for empty slice patterns, got %v", err)
	}
}

func TestCheckPathAllowed_ExactFile(t *testing.T) {
	cwd, _ := os.Getwd()
	target := filepath.Join(cwd, "pathguard.go")

	if err := checkPathAllowed(target, []string{target}); err != nil {
		t.Fatalf("expected match for exact file, got %v", err)
	}
}

func TestCheckPathAllowed_DirectoryPrefix(t *testing.T) {
	cwd, _ := os.Getwd()
	// Allow the agent package directory; pathguard.go is inside it.
	if err := checkPathAllowed(filepath.Join(cwd, "pathguard.go"), []string{cwd}); err != nil {
		t.Fatalf("expected match for dir prefix, got %v", err)
	}
}

func TestCheckPathAllowed_DirectoryPrefix_DoubleStarSuffix(t *testing.T) {
	cwd, _ := os.Getwd()
	pattern := cwd + "/**"
	if err := checkPathAllowed(filepath.Join(cwd, "pathguard.go"), []string{pattern}); err != nil {
		t.Fatalf("expected match for /** pattern, got %v", err)
	}
}

func TestCheckPathAllowed_RelativePattern(t *testing.T) {
	// "." should allow anything in the cwd.
	cwd, _ := os.Getwd()
	if err := checkPathAllowed(filepath.Join(cwd, "pathguard.go"), []string{"."}); err != nil {
		t.Fatalf("expected match for relative '.' pattern, got %v", err)
	}
}

func TestCheckPathAllowed_GlobPattern(t *testing.T) {
	cwd, _ := os.Getwd()
	pattern := filepath.Join(cwd, "*.go")
	target := filepath.Join(cwd, "pathguard.go")

	if err := checkPathAllowed(target, []string{pattern}); err != nil {
		t.Fatalf("expected glob match, got %v", err)
	}

	// A .json file must NOT match *.go
	jsonTarget := filepath.Join(cwd, "something.json")
	if err := checkPathAllowed(jsonTarget, []string{pattern}); err == nil {
		t.Fatalf("expected denial for .json against *.go glob, got nil")
	}
}

func TestCheckPathAllowed_Denied(t *testing.T) {
	allowed := []string{"/safe/dir"}
	if err := checkPathAllowed("/unsafe/secret.txt", allowed); err == nil {
		t.Fatal("expected denial for path outside allowed dirs, got nil")
	}
}

func TestCheckPathAllowed_NoPrefixFalsePositive(t *testing.T) {
	// "/safe/dir" must NOT allow "/safe/dir-other/file"
	allowed := []string{"/safe/dir"}
	if err := checkPathAllowed("/safe/dir-other/file.txt", allowed); err == nil {
		t.Fatal("expected denial: /safe/dir-other is not /safe/dir")
	}
}
