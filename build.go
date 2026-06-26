//go:build ignore

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

var skills = []struct {
	dir    string
	src    string
	output string
}{
	{"skills/build_skill", "build.go", "build"},
	{"skills/file_search", "search.go", "search"},
	{"skills/grep", "grep.go", "grep"},
	{"skills/list_directory", "list.go", "list"},
	{"skills/read_file", "read.go", "read"},
	{"skills/wiki_search", "search.go", "search"},
	{"skills/write_file", "write.go", "write"},
}

func main() {
	clean := len(os.Args) > 1 && os.Args[1] == "clean"

	for _, s := range skills {
		out := s.output
		if runtime.GOOS == "windows" {
			out += ".exe"
		}
		outPath := filepath.Join(s.dir, out)

		if clean {
			os.Remove(outPath)
			fmt.Println("removed", outPath)
			continue
		}

		cmd := exec.Command("go", "build", "-o", out, s.src)
		cmd.Dir = s.dir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "failed to build %s: %v\n", s.dir, err)
			os.Exit(1)
		}
		fmt.Println("built", outPath)
	}

	agentBin := "context-monster-cli"
	if runtime.GOOS == "windows" {
		agentBin += ".exe"
	}
	if clean {
		os.Remove(agentBin)
		fmt.Println("removed", agentBin)
		return
	}
	cmd := exec.Command("go", "build", "-o", agentBin, "./cmd/agent")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to build agent: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("built", agentBin)
}
