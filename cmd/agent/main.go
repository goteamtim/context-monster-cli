package main

import (
	"flag"
	"fmt"
	"os"

	"context-monster-cli/pkg/agent"
	"context-monster-cli/pkg/ollama"
	"context-monster-cli/pkg/personas"
	"context-monster-cli/pkg/skills"
)

const defaultSystemPrompt = "You are a helpful assistant with access to local tools. Help the user achieve their goals. " +
	"Only call a tool when you genuinely need to access the filesystem to answer the question. " +
	"For general knowledge, reasoning, or questions you can answer directly, respond with plain text and do not invoke any tools."

func main() {
	model := flag.String("model", "qwen3.5:4b", "Ollama model to use")
	skillsDir := flag.String("skills-dir", "./skills", "Directory containing skill subdirectories")
	personasDir := flag.String("personas-dir", "./personas", "Directory containing persona subdirectories")
	personaName := flag.String("persona", "", "Run a named persona by name")
	debug := flag.Bool("debug", false, "Print raw Ollama response details to stderr")
	flag.Parse()

	allSkills, err := skills.Load(*skillsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not load skills from %q: %v\n", *skillsDir, err)
		allSkills = nil
	}

	allPersonas, err := personas.Load(*personasDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not load personas from %q: %v\n", *personasDir, err)
		allPersonas = nil
	}

	var (
		activeSkills []skills.Skill
		systemPrompt string
		opts         *ollama.Options
	)

	if *personaName != "" {
		// Persona mode.
		p, found := personas.FindByName(allPersonas, *personaName)
		if !found {
			fmt.Fprintf(os.Stderr, "Error: no persona named %q found in %s\n", *personaName, *personasDir)
			os.Exit(1)
		}

		cfg := p.Manifest
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

		// Override model if persona specifies one.
		if cfg.Model != "" {
			*model = cfg.Model
		}

		// Build Ollama options from persona if set.
		if cfg.ContextWindow > 0 || cfg.MaxTokens > 0 {
			opts = &ollama.Options{
				NumCtx:     cfg.ContextWindow,
				NumPredict: cfg.MaxTokens,
			}
		}

		fmt.Fprintf(os.Stderr, "# Alias hint: alias %s='%s --persona %s'\n\n", *personaName, os.Args[0], *personaName)
		fmt.Printf("Running as persona: %s\n", *personaName)
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

	client := ollama.New("http://localhost:11434", *model, opts)
	agent.New(client, activeSkills, systemPrompt, *debug).Run()
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
