package features

import (
	"fmt"
	"regexp"
)

var (
	idPattern  = regexp.MustCompile(`^[a-z][a-z0-9-]*(\.[a-z][a-z0-9-]*)+$`)
	mvpPattern = regexp.MustCompile(`^[0-9]{4}$`)
)

var (
	validStatuses = map[Status]bool{
		StatusPlanned:         true,
		StatusInProgress:      true,
		StatusShipped:         true,
		StatusIntentionalOmit: true,
		StatusDeferred:        true,
	}
	validClasses = map[CompatibilityClass]bool{
		ClassParity:     true,
		ClassExtension:  true,
		ClassDivergence: true,
		ClassDeferred:   true,
	}
)

// Validate checks inventory structural and semantic constraints.
func Validate(inv *Inventory) error {
	if inv == nil {
		return fmt.Errorf("inventory is nil")
	}
	if inv.SchemaVersion != "1" {
		return fmt.Errorf("unsupported schema_version %q", inv.SchemaVersion)
	}
	if len(inv.Features) == 0 {
		return fmt.Errorf("inventory has no features")
	}

	seen := make(map[string]struct{}, len(inv.Features))
	for i, f := range inv.Features {
		if err := validateFeature(i, f); err != nil {
			return err
		}
		if _, dup := seen[f.ID]; dup {
			return fmt.Errorf("features[%d]: duplicate id %q", i, f.ID)
		}
		seen[f.ID] = struct{}{}
	}
	return nil
}

func validateFeature(i int, f Feature) error {
	prefix := fmt.Sprintf("features[%d]", i)

	if f.ID == "" {
		return fmt.Errorf("%s: missing id", prefix)
	}
	if !idPattern.MatchString(f.ID) {
		return fmt.Errorf("%s: invalid id %q", prefix, f.ID)
	}
	if f.Name == "" {
		return fmt.Errorf("%s (%s): missing name", prefix, f.ID)
	}
	if f.Module == "" {
		return fmt.Errorf("%s (%s): missing module", prefix, f.ID)
	}
	if !validStatuses[f.NubStatus] {
		return fmt.Errorf("%s (%s): invalid nub_status %q", prefix, f.ID, f.NubStatus)
	}
	if !validStatuses[f.MewStatus] {
		return fmt.Errorf("%s (%s): invalid mew_status %q", prefix, f.ID, f.MewStatus)
	}
	if !validClasses[f.CompatibilityClass] {
		return fmt.Errorf("%s (%s): invalid compatibility_class %q", prefix, f.ID, f.CompatibilityClass)
	}
	if f.PrimaryMVP == "" {
		return fmt.Errorf("%s (%s): missing primary_mvp", prefix, f.ID)
	}
	if !mvpPattern.MatchString(f.PrimaryMVP) {
		return fmt.Errorf("%s (%s): invalid primary_mvp %q", prefix, f.ID, f.PrimaryMVP)
	}
	if f.Tests == nil {
		return fmt.Errorf("%s (%s): missing tests array", prefix, f.ID)
	}
	return nil
}

// ValidateExtensions ensures Nub-absent features are classified as extensions.
func ValidateExtensions(inv *Inventory) error {
	for _, f := range inv.Features {
		if f.NubStatus != StatusIntentionalOmit {
			continue
		}
		if f.CompatibilityClass != ClassExtension && f.CompatibilityClass != ClassDeferred {
			return fmt.Errorf("%q has nub_status intentional-omit but compatibility_class %q", f.ID, f.CompatibilityClass)
		}
	}
	return nil
}

// ValidateMVPCoverage ensures each required MVP ID owns at least one feature.
func ValidateMVPCoverage(inv *Inventory, required []string) error {
	counts := inv.MVPCoverage()
	for _, mvp := range required {
		if counts[mvp] == 0 {
			return fmt.Errorf("MVP %s owns no inventory rows", mvp)
		}
	}
	return nil
}
