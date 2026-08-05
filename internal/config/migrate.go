package config

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/fsx"
)

// MigrationStep is one planned rewrite of a legacy key into its canonical form.
type MigrationStep struct {
	From     string // legacy dotted key as written in the file
	To       string // canonical dotted key
	OldValue any    // value as written
	NewValue any    // value after any unit/type transform
}

// MigrationPlan is the deterministic set of rewrites for one file.
// Steps are sorted by From so the same file always produces the same plan and
// the same output bytes.
type MigrationPlan struct {
	Path      string
	Steps     []MigrationStep
	Conflicts []string // canonical keys present alongside their legacy spelling
}

// Empty reports whether the plan would change nothing.
func (p MigrationPlan) Empty() bool { return len(p.Steps) == 0 }

// PlanMigration reads path and computes the rewrite plan without touching the
// file. A missing file yields an empty plan and no error, so `--check` on a
// fresh machine is not an error condition.
func PlanMigration(path string) (MigrationPlan, error) {
	plan := MigrationPlan{Path: path}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return plan, nil
		}
		return plan, apperr.Wrap(apperr.IO, "config.migrate", path, err)
	}
	parsed, err := ParseJSONC(b)
	if err != nil {
		return plan, apperr.Wrap(apperr.Config, "config.migrate", path, err)
	}
	m, ok := parsed.(map[string]any)
	if !ok {
		return plan, apperr.New(apperr.Config, "config.migrate", path, "root must be object")
	}
	return planFromFlat(path, flattenDotted(m, "")), nil
}

// planFromFlat builds the plan from an already-flattened document.
func planFromFlat(path string, flat map[string]any) MigrationPlan {
	plan := MigrationPlan{Path: path}

	keys := make([]string, 0, len(flat))
	for k := range flat {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// A canonical key that is already present blocks the rename of its legacy
	// twin: applying it would drop one of the two values.
	present := make(map[string]bool, len(flat))
	for _, k := range keys {
		present[k] = true
	}

	seenConflict := map[string]bool{}
	for _, k := range keys {
		canon := CanonicalKey(k)
		if canon == "" || canon == k {
			continue
		}
		if present[canon] {
			if !seenConflict[canon] {
				seenConflict[canon] = true
				plan.Conflicts = append(plan.Conflicts, canon)
			}
			continue
		}
		old := flat[k]
		newVal, xerr := transformMigratedValue(k, old)
		if xerr != nil {
			plan.Conflicts = append(plan.Conflicts,
				fmt.Sprintf("%s: invalid value: %v", k, xerr))
			continue
		}
		if verr := validateMigratedValue(canon, newVal); verr != nil {
			plan.Conflicts = append(plan.Conflicts,
				fmt.Sprintf("%s -> %s: invalid transformed value: %v", k, canon, verr))
			continue
		}
		plan.Steps = append(plan.Steps, MigrationStep{
			From:     k,
			To:       canon,
			OldValue: old,
			NewValue: newVal,
		})
	}
	sort.Strings(plan.Conflicts)
	return plan
}

// transformMigratedValue applies the value change a rename implies. Renames
// that only change spelling return the value untouched. Returns an error when
// the legacy value cannot be safely transformed.
func transformMigratedValue(legacyKey string, v any) (any, error) {
	switch legacyKey {
	case "network.timeout_ms":
		d, err := legacyTimeoutToDuration(v)
		if err != nil {
			return nil, err
		}
		return d.String(), nil
	}
	return v, nil
}

// legacyTimeoutToDuration converts a legacy network.timeout_ms value into a
// time.Duration. Strings, negatives, and fractions are rejected.
func legacyTimeoutToDuration(v any) (time.Duration, error) {
	var ms int64
	switch n := v.(type) {
	case int:
		ms = int64(n)
	case int64:
		ms = n
	case float64:
		if n != float64(int64(n)) {
			return 0, fmt.Errorf("network.timeout_ms must be an integer, got %v", v)
		}
		ms = int64(n)
	default:
		return 0, fmt.Errorf("network.timeout_ms must be an integer, got %T", v)
	}
	if ms < 0 {
		return 0, fmt.Errorf("network.timeout_ms must not be negative, got %d", ms)
	}
	d := time.Duration(ms) * time.Millisecond
	if d < 0 {
		return 0, fmt.Errorf("network.timeout_ms overflow: %d", ms)
	}
	return d, nil
}

// validateMigratedValue checks that a transformed value is valid for its
// canonical key per the schema registry.
func validateMigratedValue(canon string, v any) error {
	if err := validateKeyValue(canon, normalizeForWrite(v)); err != nil {
		return err
	}
	if spec := KeySpec(canon); spec != nil {
		if err := validateRange(spec, normalizeForWrite(v)); err != nil {
			return err
		}
	}
	return nil
}

// ConflictError renders the plan's conflicts as a Config error.
func (p MigrationPlan) ConflictError() error {
	if len(p.Conflicts) == 0 {
		return nil
	}
	return apperr.New(apperr.Config, "config.migrate", p.Path,
		"conflicting legacy and canonical keys: "+strings.Join(p.Conflicts, ", "))
}

// Apply rewrites the file according to the plan using comment-preserving
// splices: each step unsets the legacy key and sets the canonical one, so
// every comment outside the moved members survives. Returns the number of
// keys migrated.
func (p MigrationPlan) Apply() (int, error) {
	if err := p.ConflictError(); err != nil {
		return 0, err
	}
	if p.Empty() {
		return 0, nil
	}
	src, err := os.ReadFile(p.Path)
	if err != nil {
		return 0, apperr.Wrap(apperr.IO, "config.migrate", p.Path, err)
	}
	out := src
	for _, step := range p.Steps {
		// Set the canonical key first: if the splice fails, the file on disk
		// is still the untouched original.
		next, err := setJSONCPath(out, step.To, step.NewValue)
		if err != nil {
			return 0, apperr.Wrap(apperr.Config, "config.migrate", p.Path+":"+step.To, err)
		}
		next, changed, err := unsetJSONCPath(next, step.From)
		if err != nil {
			return 0, apperr.Wrap(apperr.Config, "config.migrate", p.Path+":"+step.From, err)
		}
		if !changed {
			return 0, apperr.New(apperr.Internal, "config.migrate", p.Path+":"+step.From,
				"planned legacy key vanished before rewrite")
		}
		out = next
	}
	// Never publish bytes we cannot read back.
	if _, err := ParseJSONC(out); err != nil {
		return 0, apperr.Wrap(apperr.Internal, "config.migrate", p.Path, err)
	}
	// Validate the complete output document against the canonical schema before
	// publishing, so a partially-migrated document never reaches disk.
	if verr := ValidateDocument(out, p.Path, ValidateOptions{}).Err(); verr != nil {
		return 0, verr
	}
	if err := fsx.PublishFileDurable(p.Path, out, 0o644); err != nil {
		return 0, apperr.Wrap(apperr.IO, "config.migrate", p.Path, err)
	}
	return len(p.Steps), nil
}
