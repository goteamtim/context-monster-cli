package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRun_success(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.txt")

	msg, code := run(WriteArgs{Path: path, Content: "hello"})
	if code != 0 {
		t.Fatalf("unexpected failure: %s", msg)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "hello" {
		t.Fatalf("unexpected file contents: %q (err: %v)", string(data), err)
	}
}

func TestRun_existingFileNoOverwrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.txt")
	_ = os.WriteFile(path, []byte("original"), 0644)

	_, code := run(WriteArgs{Path: path, Content: "new"})
	if code == 0 {
		t.Fatal("expected failure for existing file without overwrite flag")
	}
	// Original must be untouched.
	data, _ := os.ReadFile(path)
	if string(data) != "original" {
		t.Fatalf("original file was modified: got %q", string(data))
	}
}

func TestRun_overwrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.txt")
	_ = os.WriteFile(path, []byte("original"), 0644)

	_, code := run(WriteArgs{Path: path, Content: "new", Overwrite: true})
	if code != 0 {
		t.Fatal("expected success with overwrite: true")
	}
	data, _ := os.ReadFile(path)
	if string(data) != "new" {
		t.Fatalf("file was not overwritten: got %q", string(data))
	}
}

func TestRun_createsParentDirectories(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a", "b", "c", "out.txt")

	_, code := run(WriteArgs{Path: path, Content: "deep"})
	if code != 0 {
		t.Fatal("expected parent directories to be created automatically")
	}
	data, _ := os.ReadFile(path)
	if string(data) != "deep" {
		t.Fatalf("unexpected content: %q", string(data))
	}
}

func TestRun_emptyPath(t *testing.T) {
	_, code := run(WriteArgs{Path: "", Content: "hello"})
	if code == 0 {
		t.Fatal("expected failure for empty path")
	}
}

func TestRun_errorMessageNotEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.txt")
	_ = os.WriteFile(path, []byte("original"), 0644)

	msg, code := run(WriteArgs{Path: path, Content: "new"})
	if code == 0 {
		t.Fatal("expected failure")
	}
	if strings.TrimSpace(msg) == "" {
		t.Fatal("expected a non-empty error message so the agent can report it")
	}
}
