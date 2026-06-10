package personas

// PersonaManifest mirrors the persona.json file that every persona must provide.
type PersonaManifest struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	SystemPrompt string   `json:"system_prompt"`
	// Model overrides the --model flag when non-empty.
	Model        string   `json:"model,omitempty"`
	// ContextWindow sets Ollama's num_ctx option when non-zero.
	ContextWindow int     `json:"context_window,omitempty"`
	// MaxTokens sets Ollama's num_predict option when non-zero.
	MaxTokens    int      `json:"max_tokens,omitempty"`
	// Tools is the list of skill names the persona is allowed to call.
	Tools        []string `json:"tools"`
}

// Persona is a fully parsed persona with its manifest and directory path.
type Persona struct {
	Manifest PersonaManifest
	// Dir is the absolute path to the persona's directory.
	Dir string
}
