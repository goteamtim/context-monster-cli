package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

type Args struct {
	Path         string `json:"path"`
	Pattern      string `json:"pattern"`
	StartLine    int    `json:"start_line"`
	EndLine      int    `json:"end_line"`
	ContextLines int    `json:"context_lines"`
}

// maxMatches is a var so tests can override it to exercise truncation.
var maxMatches = 100

// run validates args, performs the search or range read, and returns (output, exitCode).
// On success output is the result text for stdout; on failure it is the error message.
// Keeping logic here rather than in main makes it directly testable.
func run(args Args) (string, int) {
	hasPattern := args.Pattern != ""
	hasRange := args.StartLine > 0 && args.EndLine > 0
	partialRange := (args.StartLine > 0) != (args.EndLine > 0)

	if partialRange {
		return "provide both 'start_line' and 'end_line' together", 1
	}
	if !hasPattern && !hasRange {
		return "provide 'pattern', or both 'start_line' and 'end_line'", 1
	}
	if hasRange && args.StartLine > args.EndLine {
		return "start_line must be <= end_line", 1
	}
	if args.ContextLines < 0 {
		args.ContextLines = 0
	}

	var re *regexp.Regexp
	if hasPattern {
		var err error
		re, err = regexp.Compile(args.Pattern)
		if err != nil {
			return fmt.Sprintf("invalid pattern %q: %v", args.Pattern, err), 1
		}
	}

	f, err := os.Open(args.Path)
	if err != nil {
		return fmt.Sprintf("opening %q: %v", args.Path, err), 1
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1<<20), 1<<20)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return fmt.Sprintf("reading %q: %v", args.Path, err), 1
	}

	if len(lines) == 0 {
		if hasPattern {
			return fmt.Sprintf("0 matches found for pattern %q in %s\n", args.Pattern, args.Path), 0
		}
		return "", 0
	}

	// Range-only mode: return the requested lines with line numbers.
	if !hasPattern {
		start := clamp(args.StartLine-1, 0, len(lines)-1)
		end := clamp(args.EndLine-1, 0, len(lines)-1)
		var sb strings.Builder
		for i := start; i <= end; i++ {
			fmt.Fprintf(&sb, "%d: %s\n", i+1, lines[i])
		}
		return sb.String(), 0
	}

	// Grep mode: determine search window.
	searchStart := 0
	searchEnd := len(lines) - 1
	if hasRange {
		searchStart = clamp(args.StartLine-1, 0, len(lines)-1)
		searchEnd = clamp(args.EndLine-1, 0, len(lines)-1)
	}

	// Collect matching indices within the search window.
	var matchIdxs []int
	for i := searchStart; i <= searchEnd; i++ {
		if re.MatchString(lines[i]) {
			matchIdxs = append(matchIdxs, i)
		}
	}

	total := len(matchIdxs)
	if total == 0 {
		return fmt.Sprintf("0 matches found for pattern %q in %s\n", args.Pattern, args.Path), 0
	}

	shown := total
	truncated := false
	if shown > maxMatches {
		shown = maxMatches
		truncated = true
	}

	type entry struct {
		lineNum int // 1-indexed
		isMatch bool
	}

	seen := make(map[int]bool)
	var output []entry

	for _, idx := range matchIdxs[:shown] {
		lo := clamp(idx-args.ContextLines, searchStart, idx)
		hi := clamp(idx+args.ContextLines, idx, searchEnd)

		for i := lo; i < idx; i++ {
			if !seen[i] {
				seen[i] = true
				output = append(output, entry{i + 1, false})
			}
		}
		if !seen[idx] {
			seen[idx] = true
			output = append(output, entry{idx + 1, true})
		}
		for i := idx + 1; i <= hi; i++ {
			if !seen[i] {
				seen[i] = true
				output = append(output, entry{i + 1, false})
			}
		}
	}

	var sb strings.Builder
	prev := -1
	for _, e := range output {
		if prev != -1 && e.lineNum > prev+1 {
			sb.WriteString("---\n")
		}
		if e.isMatch {
			fmt.Fprintf(&sb, "%d: %s\n", e.lineNum, lines[e.lineNum-1])
		} else {
			fmt.Fprintf(&sb, "%d  %s\n", e.lineNum, lines[e.lineNum-1])
		}
		prev = e.lineNum
	}

	if truncated {
		fmt.Fprintf(&sb, "... %d more match(es) not shown\n", total-maxMatches)
	}
	fmt.Fprintf(&sb, "%d match(es) found\n", total)
	return sb.String(), 0
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "error: missing JSON argument payload")
		os.Exit(1)
	}

	var args Args
	if err := json.Unmarshal([]byte(os.Args[1]), &args); err != nil {
		fmt.Fprintf(os.Stderr, "error parsing arguments: %v\n", err)
		os.Exit(1)
	}

	msg, code := run(args)
	if code != 0 {
		fmt.Fprintf(os.Stderr, "error: %s\n", msg)
		os.Exit(1)
	}
	fmt.Print(msg)
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
