package config

import (
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
		plan.Steps = append(plan.Steps, MigrationStep{
			From:     k,
			To:       canon,
			OldValue: old,
			NewValue: transformMigratedValue(k, old),
		})
	}
	sort.Strings(plan.Conflicts)
	return plan
}

// transformMigratedValue applies the value change a rename implies. Renames
// that only change spelling return the value untouched.
func transformMigratedValue(legacyKey string, v any) any {
	switch legacyKey {
	case "network.timeout_ms":
		// int milliseconds became a duration string.
		switch n := v.(type) {
		case int:
			return (time.Duration(n) * time.Millisecond).String()
		case float64:
			return (time.Duration(int64(n)) * time.Millisecond).String()
		}
	}
	return v
}

// conflictError renders the plan's conflicts as a Config error.
func (p MigrationPlan) conflictError() error {
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
	if err := p.conflictError(); err != nil {
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
	if err := fsx.PublishFileDurable(p.Path, out, 0o644); err != nil {
		return 0, apperr.Wrap(apperr.IO, "config.migrate", p.Path, err)
	}
	return len(p.Steps), nil
}
