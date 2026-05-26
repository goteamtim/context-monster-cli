package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type ReadArgs struct {
	Path string `json:"path"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Error: Missing JSON argument payload.")
		os.Exit(1)
	}

	var args ReadArgs
	if err := json.Unmarshal([]byte(os.Args[1]), &args); err != nil {
		fmt.Printf("Error parsing arguments: %v\n", err)
		os.Exit(1)
	}

	data, err := os.ReadFile(args.Path)
	if err != nil {
		fmt.Printf("Error reading file %q: %v\n", args.Path, err)
		os.Exit(1)
	}

	fmt.Print(string(data))
}
