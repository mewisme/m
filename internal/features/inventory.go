package features

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// Status is the implementation lifecycle state for a feature on one side.
type Status string

const (
	StatusPlanned         Status = "planned"
	StatusInProgress      Status = "in-progress"
	StatusShipped         Status = "shipped"
	StatusIntentionalOmit Status = "intentional-omit"
	StatusDeferred        Status = "deferred"
)

// CompatibilityClass classifies how Mew relates to Nub or incumbents.
type CompatibilityClass string

const (
	ClassParity     CompatibilityClass = "parity"
	ClassExtension  CompatibilityClass = "extension"
	ClassDivergence CompatibilityClass = "divergence"
	ClassDeferred   CompatibilityClass = "deferred"
)

// Feature is one inventory row.
type Feature struct {
	ID                 string             `json:"id"`
	Name               string             `json:"name"`
	Module             string             `json:"module"`
	NubStatus          Status             `json:"nub_status"`
	MewStatus          Status             `json:"mew_status"`
	CompatibilityClass CompatibilityClass `json:"compatibility_class"`
	PrimaryMVP         string             `json:"primary_mvp"`
	Tests              []string           `json:"tests"`
	Notes              string             `json:"notes,omitempty"`
}

// PublicFeature is the user-facing CLI JSON shape (no internal test IDs).
type PublicFeature struct {
	ID                 string             `json:"id"`
	Name               string             `json:"name"`
	Module             string             `json:"module"`
	NubStatus          Status             `json:"nub_status"`
	MewStatus          Status             `json:"mew_status"`
	CompatibilityClass CompatibilityClass `json:"compatibility_class"`
	PrimaryMVP         string             `json:"primary_mvp"`
}

// Inventory is the authoritative feature inventory document.
type Inventory struct {
	SchemaVersion string    `json:"schema_version"`
	Features      []Feature `json:"features"`
}

// LoadEmbedded reads the repository inventory from features/inventory.json.
func LoadEmbedded() (*Inventory, error) {
	path, err := DefaultInventoryPath()
	if err != nil {
		return nil, err
	}
	return LoadFile(path)
}

// DefaultInventoryPath locates features/inventory.json from the module root.
func DefaultInventoryPath() (string, error) {
	root, err := findModuleRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "features", "inventory.json"), nil
}

func findModuleRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("module root not found")
		}
		dir = parent
	}
}

// LoadFile reads inventory from a filesystem path.
func LoadFile(path string) (*Inventory, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(b)
}

// Parse decodes and validates inventory JSON.
func Parse(data []byte) (*Inventory, error) {
	var inv Inventory
	if err := json.Unmarshal(data, &inv); err != nil {
		return nil, fmt.Errorf("decode inventory: %w", err)
	}
	if err := Validate(&inv); err != nil {
		return nil, err
	}
	return &inv, nil
}

// Filter returns features matching optional module and mew_status filters.
func (inv *Inventory) Filter(module, mewStatus string) []Feature {
	out := make([]Feature, 0, len(inv.Features))
	for _, f := range inv.Features {
		if module != "" && f.Module != module {
			continue
		}
		if mewStatus != "" && string(f.MewStatus) != mewStatus {
			continue
		}
		out = append(out, f)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Module != out[j].Module {
			return out[i].Module < out[j].Module
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// PublicView strips internal fields for CLI output.
func PublicView(features []Feature) []PublicFeature {
	out := make([]PublicFeature, len(features))
	for i, f := range features {
		out[i] = PublicFeature{
			ID:                 f.ID,
			Name:               f.Name,
			Module:             f.Module,
			NubStatus:          f.NubStatus,
			MewStatus:          f.MewStatus,
			CompatibilityClass: f.CompatibilityClass,
			PrimaryMVP:         f.PrimaryMVP,
		}
	}
	return out
}

// MVPCoverage returns primary_mvp values present in the inventory.
func (inv *Inventory) MVPCoverage() map[string]int {
	counts := make(map[string]int)
	for _, f := range inv.Features {
		counts[f.PrimaryMVP]++
	}
	return counts
}
