package app

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/mewisme/mew/internal/apperr"
)

// GenerationBinding is the project install generation fingerprint used by bin metadata.
type GenerationBinding struct {
	GenerationID string `json:"generationID"`
	Fingerprint  string `json:"fingerprint"`
}

const generationRel = ".mew/generation.json"

// LoadGenerationBinding reads .mew/generation.json when present.
func LoadGenerationBinding(projectRoot string) (GenerationBinding, error) {
	var empty GenerationBinding
	path := filepath.Join(projectRoot, filepath.FromSlash(generationRel))
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return empty, nil
		}
		return empty, apperr.Wrap(apperr.IO, "app.generation", path, err)
	}
	var doc GenerationBinding
	if err := json.Unmarshal(b, &doc); err != nil {
		return empty, apperr.Wrap(apperr.Integrity, "app.generation", path, err)
	}
	return doc, nil
}

// WriteGenerationBinding stages generation binding for transactional publish.
func WriteGenerationBinding(projectRoot string, bind GenerationBinding) error {
	path := filepath.Join(projectRoot, ".mew", "generation.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return apperr.Wrap(apperr.IO, "app.generation", path, err)
	}
	b, err := json.MarshalIndent(bind, "", "  ")
	if err != nil {
		return apperr.Wrap(apperr.Internal, "app.generation", path, err)
	}
	b = append(b, '\n')
	return os.WriteFile(path, b, 0o644)
}
