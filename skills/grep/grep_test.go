package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeTemp creates a temp file with the given content and returns its path.
func writeTemp(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "grep_test_*.txt")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.WriteString(content)
	_ = f.Close()
	return f.Name()
}

// --- Grep mode ---

func TestRun_grepBasicMatch(t *testing.T) {
	path := writeTemp(t, "apple\nbanana\napricot\n")
	out, code := run(Args{Path: path, Pattern: "^ap"})
	if code != 0 {
		t.Fatalf("unexpected exit code %d: %s", code, out)
	}
	if !strings.Contains(out, "1: apple") {
		t.Errorf("expected line 1 match, got:\n%s", out)
	}
	if !strings.Contains(out, "3: apricot") {
		t.Errorf("expected line 3 match, got:\n%s", out)
	}
	if strings.Contains(out, "banana") {
		t.Errorf("non-matching line should not appear, got:\n%s", out)
	}
	if !strings.Contains(out, "2 match(es) found") {
		t.Errorf("expected match count footer, got:\n%s", out)
	}
}

func TestRun_grepNoMatches(t *testing.T) {
	path := writeTemp(t, "apple\nbanana\n")
	out, code := run(Args{Path: path, Pattern: "mango"})
	if code != 0 {
		t.Fatalf("unexpected exit code %d: %s", code, out)
	}
	if !strings.Contains(out, "0 matches") {
		t.Errorf("expected zero-match message, got:\n%s", out)
	}
}

func TestRun_grepInvalidRegex(t *testing.T) {
	path := writeTemp(t, "anything\n")
	_, code := run(Args{Path: path, Pattern: "[invalid"})
	if code == 0 {
		t.Fatal("expected non-zero exit for invalid regex")
	}
}

func TestRun_grepContextLines(t *testing.T) {
	path := writeTemp(t, "before\nmatch\nafter\n")
	out, code := run(Args{Path: path, Pattern: "match", ContextLines: 1})
	if code != 0 {
		t.Fatalf("unexpected exit code %d: %s", code, out)
	}
	if !strings.Contains(out, "1  before") {
		t.Errorf("expected context line 'before', got:\n%s", out)
	}
	if !strings.Contains(out, "2: match") {
		t.Errorf("expected match line, got:\n%s", out)
	}
	if !strings.Contains(out, "3  after") {
		t.Errorf("expected context line 'after', got:\n%s", out)
	}
}

func TestRun_grepSeparatorBetweenDisjointMatches(t *testing.T) {
	// match1 at line 1, match2 at line 7 — no context, so a separator must appear.
	path := writeTemp(t, "match1\na\nb\nc\nd\ne\nmatch2\n")
	out, code := run(Args{Path: path, Pattern: "match", ContextLines: 0})
	if code != 0 {
		t.Fatalf("unexpected exit code %d: %s", code, out)
	}
	if !strings.Contains(out, "---") {
		t.Errorf("expected --- separator between disjoint matches, got:\n%s", out)
	}
}

func TestRun_grepNoSeparatorWhenContextOverlaps(t *testing.T) {
	// Two matches on lines 2 and 4 with context=1 — the context lines merge.
	path := writeTemp(t, "a\nmatch1\nb\nmatch2\nc\n")
	out, code := run(Args{Path: path, Pattern: "match", ContextLines: 1})
	if code != 0 {
		t.Fatalf("unexpected exit code %d: %s", code, out)
	}
	if strings.Contains(out, "---") {
		t.Errorf("did not expect separator when context merges, got:\n%s", out)
	}
}

func TestRun_grepNegativeContextClampedToZero(t *testing.T) {
	path := writeTemp(t, "before\nmatch\nafter\n")
	out, code := run(Args{Path: path, Pattern: "match", ContextLines: -5})
	if code != 0 {
		t.Fatalf("negative context_lines should not cause an error, got code %d: %s", code, out)
	}
	if strings.Contains(out, "before") || strings.Contains(out, "after") {
		t.Errorf("negative context_lines should be treated as 0, got:\n%s", out)
	}
}

func TestRun_grepTruncation(t *testing.T) {
	orig := maxMatches
	maxMatches = 3
	defer func() { maxMatches = orig }()

	var content strings.Builder
	for i := 1; i <= 5; i++ {
		fmt.Fprintf(&content, "match%d\n", i)
	}
	path := writeTemp(t, content.String())

	out, code := run(Args{Path: path, Pattern: "match"})
	if code != 0 {
		t.Fatalf("unexpected exit code %d: %s", code, out)
	}
	if !strings.Contains(out, "... 2 more match(es) not shown") {
		t.Errorf("expected truncation footer, got:\n%s", out)
	}
	if !strings.Contains(out, "5 match(es) found") {
		t.Errorf("expected total count in footer even when truncated, got:\n%s", out)
	}
}

// --- Range-only mode ---

func TestRun_rangeOnly(t *testing.T) {
	path := writeTemp(t, "line1\nline2\nline3\nline4\nline5\n")
	out, code := run(Args{Path: path, StartLine: 2, EndLine: 4})
	if code != 0 {
		t.Fatalf("unexpected exit code %d: %s", code, out)
	}
	if strings.Contains(out, "1:") || strings.Contains(out, "5:") {
		t.Errorf("out-of-range lines should not appear, got:\n%s", out)
	}
	for _, want := range []string{"2: line2", "3: line3", "4: line4"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in output, got:\n%s", want, out)
		}
	}
}

func TestRun_rangeClampsToBounds(t *testing.T) {
	path := writeTemp(t, "only\n")
	out, code := run(Args{Path: path, StartLine: 1, EndLine: 100})
	if code != 0 {
		t.Fatalf("unexpected exit code %d: %s", code, out)
	}
	if !strings.Contains(out, "1: only") {
		t.Errorf("expected clamped output, got:\n%s", out)
	}
}

func TestRun_emptyFile(t *testing.T) {
	path := writeTemp(t, "")
	// Range mode on empty file should not panic.
	_, code := run(Args{Path: path, StartLine: 1, EndLine: 5})
	if code != 0 {
		t.Fatal("empty file range read should succeed silently")
	}
	// Grep mode on empty file should report 0 matches.
	out, code := run(Args{Path: path, Pattern: "x"})
	if code != 0 {
		t.Fatal("empty file grep should succeed with 0 matches")
	}
	if !strings.Contains(out, "0 matches") {
		t.Errorf("expected 0 matches message, got:\n%s", out)
	}
}

// --- Combined mode ---

func TestRun_combined(t *testing.T) {
	// Lines: 1=match, 2=ignore, 3=match, 4=ignore, 5=match
	// Search window 1-3: only lines 1 and 3 match.
	path := writeTemp(t, "match\nignore\nmatch\nignore\nmatch\n")
	out, code := run(Args{Path: path, Pattern: "match", StartLine: 1, EndLine: 3})
	if code != 0 {
		t.Fatalf("unexpected exit code %d: %s", code, out)
	}
	if !strings.Contains(out, "1: match") || !strings.Contains(out, "3: match") {
		t.Errorf("expected matches on lines 1 and 3, got:\n%s", out)
	}
	if !strings.Contains(out, "2 match(es) found") {
		t.Errorf("line 5 is outside the range and must not be counted, got:\n%s", out)
	}
}

// --- Error cases ---

func TestRun_errorMissingBoth(t *testing.T) {
	path := writeTemp(t, "anything\n")
	_, code := run(Args{Path: path})
	if code == 0 {
		t.Fatal("expected error when neither pattern nor range is provided")
	}
}

func TestRun_errorPartialRange(t *testing.T) {
	path := writeTemp(t, "anything\n")
	_, code := run(Args{Path: path, StartLine: 1})
	if code == 0 {
		t.Fatal("expected error for start_line without end_line")
	}
	_, code = run(Args{Path: path, EndLine: 5})
	if code == 0 {
		t.Fatal("expected error for end_line without start_line")
	}
}

func TestRun_errorStartGreaterThanEnd(t *testing.T) {
	path := writeTemp(t, "anything\n")
	_, code := run(Args{Path: path, StartLine: 5, EndLine: 2})
	if code == 0 {
		t.Fatal("expected error when start_line > end_line")
	}
}

func TestRun_errorFileNotFound(t *testing.T) {
	_, code := run(Args{Path: filepath.Join(t.TempDir(), "does_not_exist.txt"), Pattern: "x"})
	if code == 0 {
		t.Fatal("expected error for missing file")
	}
}
