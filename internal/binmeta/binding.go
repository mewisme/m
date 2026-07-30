package binmeta

import "encoding/json"

// GenerationBinding binds filesystem state to published bin metadata.
type GenerationBinding struct {
	GenerationID string `json:"generationID"`
	Fingerprint  string `json:"fingerprint"`
}

// DecodeGenerationBinding parses generation.json bytes.
func DecodeGenerationBinding(b []byte) (GenerationBinding, error) {
	var bind GenerationBinding
	if err := json.Unmarshal(b, &bind); err != nil {
		return GenerationBinding{}, err
	}
	return bind, nil
}

// CommandIndex is the canonical command ownership index from bins metadata.
type CommandIndex Document
