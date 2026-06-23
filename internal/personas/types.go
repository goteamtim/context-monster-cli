package personas

// PersonaManifest mirrors the persona.json file that every persona must provide.
type PersonaManifest struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	SystemPrompt string `json:"system_prompt"`
	// Model overrides the --model flag when non-empty.
	Model string `json:"model,omitempty"`
	// ContextWindow sets Ollama's num_ctx option when non-zero.
	ContextWindow int `json:"context_window,omitempty"`
	// MaxTokens sets Ollama's num_predict option when non-zero.
	MaxTokens int `json:"max_tokens,omitempty"`
	// Tools is the list of skill names the persona is allowed to call.
	Tools []string `json:"tools"`
	// AllowedPaths is an optional list of file/directory patterns the persona
	// may access via tools. Supports exact paths, directory prefixes, and glob
	// patterns (filepath.Match syntax). An empty list means no restrictions.
	AllowedPaths []string `json:"allowed_paths,omitempty"`
	// Record enables episode logging when this persona is active.
	// Episodes are appended to <persona_dir>/training/episodes.jsonl.
	Record bool `json:"record,omitempty"`
}

// Persona is a fully parsed persona with its manifest and directory path.
type Persona struct {
	Manifest PersonaManifest
	// Dir is the absolute path to the persona's directory.
	Dir string
}
