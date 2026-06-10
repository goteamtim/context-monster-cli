package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type WriteArgs struct {
	Path      string `json:"path"`
	Content   string `json:"content"`
	Overwrite bool   `json:"overwrite"`
	Append    bool   `json:"append"`
}

// run performs the write and returns a message and an exit code.
// Keeping logic here (rather than in main) makes it directly testable.
func run(args WriteArgs) (string, int) {
	if args.Path == "" {
		return "path is required", 1
	}

	if err := os.MkdirAll(filepath.Dir(args.Path), 0755); err != nil {
		return fmt.Sprintf("creating directories for %q: %v", args.Path, err), 1
	}

	// Append mode: open with O_APPEND|O_CREATE, never touches existing content.
	if args.Append {
		f, err := os.OpenFile(args.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Sprintf("opening file %q for append: %v", args.Path, err), 1
		}
		defer f.Close()
		n, err := f.WriteString(args.Content)
		if err != nil {
			return fmt.Sprintf("appending to file %q: %v", args.Path, err), 1
		}
		return fmt.Sprintf("Successfully appended %d bytes to %q.", n, args.Path), 0
	}

	// Guard against unintentional overwrites.
	if _, err := os.Stat(args.Path); err == nil && !args.Overwrite {
		return fmt.Sprintf("file %q already exists. Pass overwrite: true to replace it.", args.Path), 1
	}

	if err := os.WriteFile(args.Path, []byte(args.Content), 0644); err != nil {
		return fmt.Sprintf("writing file %q: %v", args.Path, err), 1
	}

	return fmt.Sprintf("Successfully wrote %d bytes to %q.", len(args.Content), args.Path), 0
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Error: missing JSON argument payload")
		os.Exit(1)
	}

	var args WriteArgs
	if err := json.Unmarshal([]byte(os.Args[1]), &args); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing arguments: %v\n", err)
		os.Exit(1)
	}

	msg, code := run(args)
	if code != 0 {
		fmt.Fprintf(os.Stderr, "Error: %s\n", msg)
		os.Exit(1)
	}
	fmt.Println(msg)
}
