package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type WriteArgs struct {
	Path      string `json:"path"`
	Content   string `json:"content"`
	Overwrite bool   `json:"overwrite"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Error: Missing JSON argument payload.")
		os.Exit(1)
	}

	var args WriteArgs
	if err := json.Unmarshal([]byte(os.Args[1]), &args); err != nil {
		fmt.Printf("Error parsing arguments: %v\n", err)
		os.Exit(1)
	}

	if args.Path == "" {
		fmt.Println("Error: path is required.")
		os.Exit(1)
	}

	// Check if file exists and guard against unintentional overwrites.
	if _, err := os.Stat(args.Path); err == nil && !args.Overwrite {
		fmt.Printf("Error: file %q already exists. Pass overwrite: true to replace it.\n", args.Path)
		os.Exit(1)
	}

	if err := os.WriteFile(args.Path, []byte(args.Content), 0644); err != nil {
		fmt.Printf("Error writing file %q: %v\n", args.Path, err)
		os.Exit(1)
	}

	fmt.Printf("Successfully wrote %d bytes to %q.\n", len(args.Content), args.Path)
}
