package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type SearchArgs struct {
	Dir string `json:"dir"`
	Ext string `json:"ext"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Error: Missing JSON argument payload parameter.")
		os.Exit(1)
	}

	var args SearchArgs
	if err := json.Unmarshal([]byte(os.Args[1]), &args); err != nil {
		fmt.Printf("Error parsing execution arguments: %v\n", err)
		os.Exit(1)
	}

	if !strings.HasPrefix(args.Ext, ".") {
		args.Ext = "." + args.Ext
	}

	var matches []string

	err := filepath.Walk(args.Dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && strings.HasPrefix(info.Name(), ".") && info.Name() != "." {
			return filepath.SkipDir
		}
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), args.Ext) {
			matches = append(matches, path)
		}
		return nil
	})

	if err != nil {
		fmt.Printf("Error searching directory: %v\n", err)
		os.Exit(1)
	}

	if len(matches) == 0 {
		fmt.Printf("No files matching extension '%s' found in '%s'.\n", args.Ext, args.Dir)
		return
	}

	fmt.Println("Found the following matching files:\n" + strings.Join(matches, "\n"))
}
