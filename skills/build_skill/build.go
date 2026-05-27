package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type BuildArgs struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Parameters  string `json:"parameters"`
	Language    string `json:"language"`
	Code        string `json:"code"`
}

// manifest is the shape written to manifest.json.
type manifest struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
	Command     string          `json:"command"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Error: missing JSON argument payload.")
		os.Exit(1)
	}

	var args BuildArgs
	if err := json.Unmarshal([]byte(os.Args[1]), &args); err != nil {
		fmt.Printf("Error parsing arguments: %v\n", err)
		os.Exit(1)
	}

	if args.Name == "" || args.Language == "" || args.Code == "" {
		fmt.Println("Error: name, language, and code are required.")
		os.Exit(1)
	}

	// Validate that parameters is valid JSON before writing it into the manifest.
	if !json.Valid([]byte(args.Parameters)) {
		fmt.Printf("Error: 'parameters' is not valid JSON: %s\n", args.Parameters)
		os.Exit(1)
	}

	// Prefer the env var injected by the agent (works correctly under go run).
	// Fall back to resolving from the binary path for direct invocations.
	skillsRoot := os.Getenv("CM_SKILLS_DIR")
	if skillsRoot == "" {
		exe, err := os.Executable()
		if err != nil {
			fmt.Printf("Error resolving executable path: %v\n", err)
			os.Exit(1)
		}
		// skills/build_skill/build  ->  skills/<name>
		skillsRoot = filepath.Dir(filepath.Dir(exe))
	}
	skillDir := filepath.Join(skillsRoot, args.Name)

	if err := os.MkdirAll(skillDir, 0755); err != nil {
		fmt.Printf("Error creating skill directory %q: %v\n", skillDir, err)
		os.Exit(1)
	}

	// Determine command and source filename by language.
	var command, sourceFile string
	switch args.Language {
	case "go":
		command = "./" + args.Name
		sourceFile = "main.go"
	case "python":
		command = "python3 run.py"
		sourceFile = "run.py"
	case "bash":
		command = "bash run.sh"
		sourceFile = "run.sh"
	default:
		fmt.Printf("Error: unsupported language %q (supported: go, python, bash)\n", args.Language)
		os.Exit(1)
	}

	// Write manifest.json.
	m := manifest{
		Name:        args.Name,
		Description: args.Description,
		Parameters:  json.RawMessage(args.Parameters),
		Command:     command,
	}
	manifestData, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		fmt.Printf("Error serialising manifest: %v\n", err)
		os.Exit(1)
	}
	manifestPath := filepath.Join(skillDir, "manifest.json")
	if err := os.WriteFile(manifestPath, manifestData, 0644); err != nil {
		fmt.Printf("Error writing manifest: %v\n", err)
		os.Exit(1)
	}

	// Write source file.
	sourcePath := filepath.Join(skillDir, sourceFile)
	if err := os.WriteFile(sourcePath, []byte(args.Code), 0644); err != nil {
		fmt.Printf("Error writing source file: %v\n", err)
		os.Exit(1)
	}

	// Language-specific post-processing.
	switch args.Language {
	case "go":
		fmt.Printf("Compiling Go skill %q...\n", args.Name)
		binaryPath := filepath.Join(skillDir, args.Name)
		cmd := exec.Command("go", "build", "-o", binaryPath, sourcePath)
		cmd.Dir = skillDir
		out, buildErr := cmd.CombinedOutput()
		if buildErr != nil {
			fmt.Printf("Skill files written but compilation failed:\n%s\nFix the code and run:\n  cd %s && go build -o %s main.go\n",
				string(out), skillDir, args.Name)
			os.Exit(1)
		}
		fmt.Printf("Skill %q built successfully at %s\n", args.Name, binaryPath)

	case "bash":
		if err := os.Chmod(sourcePath, 0755); err != nil {
			fmt.Printf("Warning: could not chmod run.sh: %v\n", err)
		}
		fmt.Printf("Skill %q created at %s\n", args.Name, skillDir)

	case "python":
		fmt.Printf("Skill %q created at %s\n", args.Name, skillDir)
	}

	fmt.Printf("Restart the agent to load the new skill.\n")
}
