package personas

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Load scans dir for subdirectories containing a persona.json and returns all
// parsed personas. Subdirectories without a persona.json are silently skipped.
func Load(dir string) ([]Persona, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var result []Persona
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		personaDir := filepath.Join(dir, entry.Name())
		manifestPath := filepath.Join(personaDir, "persona.json")
		data, err := os.ReadFile(manifestPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("reading persona manifest %s: %w", manifestPath, err)
		}
		var m PersonaManifest
		if err := json.Unmarshal(data, &m); err != nil {
			return nil, fmt.Errorf("parsing persona manifest %s: %w", manifestPath, err)
		}
		agentsPath := filepath.Join(personaDir, "AGENTS.md")
		if agentsContent, err := os.ReadFile(agentsPath); err == nil {
			m.SystemPrompt = string(agentsContent)
		}
		result = append(result, Persona{Manifest: m, Dir: personaDir})
	}
	return result, nil
}

// FindByName returns the first persona with the given name, or (Persona{}, false).
func FindByName(personaList []Persona, name string) (Persona, bool) {
	for _, p := range personaList {
		if p.Manifest.Name == name {
			return p, true
		}
	}
	return Persona{}, false
}
