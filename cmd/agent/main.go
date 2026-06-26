package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/goteamtim/context-monster-cli/internal/agent"
	"github.com/goteamtim/context-monster-cli/internal/ollama"
	"github.com/goteamtim/context-monster-cli/internal/personas"
	"github.com/goteamtim/context-monster-cli/internal/skills"
	"github.com/goteamtim/context-monster-cli/internal/training"
)

const defaultSystemPrompt = "You are a helpful assistant with access to local tools. Help the user achieve their goals. " +
	"Only call a tool when you genuinely need to access the filesystem to answer the question. " +
	"For general knowledge, reasoning, or questions you can answer directly, respond with plain text and do not invoke any tools."

const version = "0.1.0"

func main() {
	model := flag.String("model", "qwen3.5:4b", "Ollama model to use")
	host := flag.String("host", "http://localhost:11434", "Ollama base URL")
	contextWindow := flag.Int("context-window", 8192, "Context window size in tokens; used for automatic history compaction")
	skillsDir := flag.String("skills-dir", "./skills", "Directory containing skill subdirectories")
	personasDir := flag.String("personas-dir", "./personas", "Directory containing persona subdirectories")
	personaName := flag.String("persona", "", "Run a named persona by name")
	debug := flag.Bool("debug", false, "Print raw Ollama response details to stderr")
	record := flag.Bool("record", false, "Record episodes to training/episodes.jsonl for the active persona (or ./training/ if no persona)")
	ver := flag.Bool("version", false, "Print version and exit")
	flag.Parse()

	if *ver {
		fmt.Println("context-monster-cli v" + version)
		return
	}

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
		activeMeta   training.TrajectoryMetadata
		trainingDir  string
	)

	activeMeta = training.TrajectoryMetadata{
		Model:         *model,
		Provider:      "ollama",
		ContextWindow: *contextWindow,
	}

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

		// Populate episode metadata with persona-level values.
		activeMeta.Model = *model
		activeMeta.PersonaName = cfg.Name
		if cfg.ContextWindow > 0 {
			activeMeta.ContextWindow = cfg.ContextWindow
		}

		// Enable recording if the --record flag is set or the persona opts in.
		if *record || cfg.Record {
			trainingDir = filepath.Join(p.Dir, "training")
		}

		fmt.Fprintf(os.Stderr, "# Alias hint: alias %s='%s --persona %s'\n\n", *personaName, os.Args[0], *personaName)
		fmt.Printf("Running as persona: %s\n", *personaName)
		if len(activeSkills) > 0 {
			names := make([]string, len(activeSkills))
			for i, s := range activeSkills {
				names[i] = s.Manifest.Name
			}
			fmt.Printf("Tools available: %s\n", strings.Join(names, ", "))
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
			fmt.Printf("Loaded %d skill(s): %s\n", len(activeSkills), strings.Join(names, ", "))
		} else {
			fmt.Println("No skills loaded (running without tools).")
		}

		// Enable recording if --record is set (no persona dir, use ./training).
		if *record {
			trainingDir = "./training"
		}
	}

	client := ollama.New(*host, *model, opts)

	var allowedPaths []string
	if *personaName != "" {
		p, _ := personas.FindByName(allPersonas, *personaName)
		allowedPaths = p.Manifest.AllowedPaths
	}

	var logger *training.Logger
	if trainingDir != "" {
		var logErr error
		logger, logErr = training.New(trainingDir)
		if logErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not initialise episode logger: %v\n", logErr)
		} else {
			fmt.Fprintf(os.Stderr, "Recording trajectories to: %s/trajectories.jsonl\n", trainingDir)
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	agent.New(client, activeSkills, systemPrompt, *debug, allowedPaths, logger, activeMeta).Run(ctx)
}
