package skills

import "context-monster-cli/pkg/ollama"

// ParamDef describes a single parameter as declared in a manifest.
type ParamDef struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

// Parameters is the JSON Schema-style parameters block from a manifest.
type Parameters struct {
	Type       string               `json:"type"`
	Properties map[string]ParamDef  `json:"properties"`
	Required   []string             `json:"required"`
}

// StandaloneConfig is the optional block that makes a skill runnable as a
// self-contained persona via --skill <name>.
type StandaloneConfig struct {
	// SystemPrompt is injected as the first system message when the persona starts.
	SystemPrompt string `json:"system_prompt"`
	// Tools is the curated list of skill names the persona is allowed to call.
	// An empty slice means the persona has no tool access.
	Tools []string `json:"tools"`
}

// Manifest mirrors the manifest.json file that every skill must provide.
type Manifest struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Parameters  Parameters `json:"parameters"`
	// Command is the executable + optional args, e.g. "./search" or "python3 run.py".
	// Paths starting with "." are resolved relative to the skill's directory.
	// May be empty for pure-persona skills that are never called as a tool.
	Command    string            `json:"command"`
	Standalone *StandaloneConfig `json:"standalone,omitempty"`
}

// Skill is a fully parsed, ready-to-use skill with its manifest and directory path.
type Skill struct {
	Manifest Manifest
	// Dir is the absolute path to the skill's directory.
	Dir string
}

// ToOllamaTool converts the skill into the Tool format expected by the Ollama API.
func (s Skill) ToOllamaTool() ollama.Tool {
	props := make(map[string]ollama.ToolFunctionParam, len(s.Manifest.Parameters.Properties))
	for k, v := range s.Manifest.Parameters.Properties {
		props[k] = ollama.ToolFunctionParam{
			Type:        v.Type,
			Description: v.Description,
		}
	}
	return ollama.Tool{
		Type: "function",
		Function: ollama.ToolFunction{
			Name:        s.Manifest.Name,
			Description: s.Manifest.Description,
			Parameters: ollama.ToolFunctionParameters{
				Type:       s.Manifest.Parameters.Type,
				Properties: props,
				Required:   s.Manifest.Parameters.Required,
			},
		},
	}
}
