package main

import (
	"flag"
	"fmt"
	"os"

	"context-monster-cli/pkg/agent"
	"context-monster-cli/pkg/ollama"
	"context-monster-cli/pkg/skills"
)

const defaultSystemPrompt = "You are a helpful assistant with access to local tools. Help the user achieve their goals. " +
	"Only call a tool when you genuinely need to access the filesystem to answer the question. " +
	"For general knowledge, reasoning, or questions you can answer directly, respond with plain text and do not invoke any tools."

func main() {
	model := flag.String("model", "qwen3.5:4b", "Ollama model to use")
	skillsDir := flag.String("skills-dir", "./skills", "Directory containing skill subdirectories")
	skillName := flag.String("skill", "", "Run a standalone persona skill by name")
	debug := flag.Bool("debug", false, "Print raw Ollama response details to stderr")
	flag.Parse()

	allSkills, err := skills.Load(*skillsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not load skills from %q: %v\n", *skillsDir, err)
		allSkills = nil
	}

	var (
		activeSkills []skills.Skill
		systemPrompt string
	)

	if *skillName != "" {
		// Standalone persona mode.
		persona, found := skills.FindByName(allSkills, *skillName)
		if !found {
			fmt.Fprintf(os.Stderr, "Error: no skill named %q found in %s\n", *skillName, *skillsDir)
			os.Exit(1)
		}
		if persona.Manifest.Standalone == nil {
			fmt.Fprintf(os.Stderr, "Error: skill %q does not have a 'standalone' block in its manifest\n", *skillName)
			os.Exit(1)
		}

		cfg := persona.Manifest.Standalone
		systemPrompt = cfg.SystemPrompt

		// Filter allSkills down to the curated list declared by the persona.
		allowed := make(map[string]bool, len(cfg.Tools))
		for _, t := range cfg.Tools {
			allowed[t] = true
		}
		for _, s := range allSkills {
			if allowed[s.Manifest.Name] {
				activeSkills = append(activeSkills, s)
			}
		}

		fmt.Fprintf(os.Stderr, "# Alias hint: alias %s='context-monster-cli --skill %s'\n\n", *skillName, *skillName)
		fmt.Printf("Running as persona: %s\n", *skillName)
		if len(activeSkills) > 0 {
			names := make([]string, len(activeSkills))
			for i, s := range activeSkills {
				names[i] = s.Manifest.Name
			}
			fmt.Printf("Tools available: %s\n", joinStrings(names, ", "))
		} else {
			fmt.Println("Tools available: none")
		}
	} else {
		// Normal mode.
		activeSkills = allSkills
		systemPrompt = defaultSystemPrompt
		if len(activeSkills) > 0 {
			names := make([]string, len(activeSkills))
			for i, s := range activeSkills {
				names[i] = s.Manifest.Name
			}
			fmt.Printf("Loaded %d skill(s): %s\n", len(activeSkills), joinStrings(names, ", "))
		} else {
			fmt.Println("No skills loaded (running without tools).")
		}
	}

	client := ollama.New("http://localhost:11434", *model)
	a := agent.New(client, activeSkills, systemPrompt, *debug)
	a.Run()
}

func joinStrings(ss []string, sep string) string {
	result := ""
	for i, s := range ss {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}

