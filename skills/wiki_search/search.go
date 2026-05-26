package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type args struct {
	WikiDir string `json:"wiki_dir"`
	Query   string `json:"query"`
}

type entry struct {
	title   string
	relPath string
	summary string
	score   int
}

// indexLineRe matches lines of the form:  - [Title](path) — summary
// The dash separator may be an em dash (—) or a plain hyphen-minus (-).
var indexLineRe = regexp.MustCompile(`^\s*-\s+\[([^\]]+)\]\(([^)]+)\)(?:\s+[—–-]+\s+(.*))?$`)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: search <json-args>")
		os.Exit(1)
	}

	var a args
	if err := json.Unmarshal([]byte(os.Args[1]), &a); err != nil {
		fmt.Fprintf(os.Stderr, "invalid args: %v\n", err)
		os.Exit(1)
	}

	if a.WikiDir == "" || a.Query == "" {
		fmt.Fprintln(os.Stderr, "wiki_dir and query are required")
		os.Exit(1)
	}

	indexPath := filepath.Join(a.WikiDir, "index.md")
	indexData, err := os.ReadFile(indexPath)
	if err != nil {
		fmt.Println("Wiki not initialized. Create index.md inside", a.WikiDir, "to start.")
		return
	}

	keywords := strings.Fields(strings.ToLower(a.Query))
	var entries []entry

	for _, line := range strings.Split(string(indexData), "\n") {
		m := indexLineRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		e := entry{
			title:   m[1],
			relPath: m[2],
			summary: m[3],
		}
		haystack := strings.ToLower(e.title + " " + e.summary)
		for _, kw := range keywords {
			if strings.Contains(haystack, kw) {
				e.score++
			}
		}
		if e.score > 0 {
			entries = append(entries, e)
		}
	}

	if len(entries) == 0 {
		fmt.Println("No relevant pages found.")
		return
	}

	// Sort descending by score (simple insertion sort — N is tiny)
	for i := 1; i < len(entries); i++ {
		for j := i; j > 0 && entries[j].score > entries[j-1].score; j-- {
			entries[j], entries[j-1] = entries[j-1], entries[j]
		}
	}

	// Cap at top 5
	if len(entries) > 5 {
		entries = entries[:5]
	}

	var sb strings.Builder
	for i, e := range entries {
		pagePath := filepath.Join(a.WikiDir, e.relPath)
		content, err := os.ReadFile(pagePath)
		if err != nil {
			fmt.Fprintf(&sb, "## %s\n(page not found at %s)\n---\n", e.title, pagePath)
			continue
		}
		if i > 0 {
			sb.WriteString("\n")
		}
		fmt.Fprintf(&sb, "## %s\n%s\n---\n", e.title, strings.TrimSpace(string(content)))
	}

	fmt.Print(sb.String())
}
