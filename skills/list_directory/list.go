package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type ListArgs struct {
	Path string `json:"path"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Error: missing JSON argument payload")
		os.Exit(1)
	}

	var args ListArgs
	if err := json.Unmarshal([]byte(os.Args[1]), &args); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing arguments: %v\n", err)
		os.Exit(1)
	}

	entries, err := os.ReadDir(args.Path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing directory %q: %v\n", args.Path, err)
		os.Exit(1)
	}

	if len(entries) == 0 {
		fmt.Printf("Directory %q is empty.\n", args.Path)
		return
	}

	lines := make([]string, len(entries))
	for i, entry := range entries {
		kind := "file"
		if entry.IsDir() {
			kind = "dir"
		}
		lines[i] = fmt.Sprintf("[%s] %s", kind, entry.Name())
	}

	fmt.Println(strings.Join(lines, "\n"))
}
