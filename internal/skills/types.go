package skills

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

// Manifest mirrors the manifest.json file that every skill must provide.
type Manifest struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Parameters  Parameters `json:"parameters"`
	// Command is the executable + optional args, e.g. "./search" or "python3 run.py".
	// Paths starting with "." are resolved relative to the skill's directory.
	Command string `json:"command"`
}

// Skill is a fully parsed, ready-to-use skill with its manifest and directory path.
type Skill struct {
	Manifest Manifest
	// Dir is the absolute path to the skill's directory.
	Dir string
}


