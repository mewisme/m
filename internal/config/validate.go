package config

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
)

// Severity classifies a validation diagnostic.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

// DiagnosticCode identifies a validation failure class so callers can react to
// the kind of problem without matching on message text.
type DiagnosticCode string

const (
	DiagSyntax         DiagnosticCode = "syntax"          // JSONC does not parse
	DiagRoot           DiagnosticCode = "root"            // root is not an object
	DiagRead           DiagnosticCode = "read"            // file cannot be read
	DiagUnknownKey     DiagnosticCode = "unknown_key"     // key is not in the registry
	DiagType           DiagnosticCode = "type"            // value has the wrong type
	DiagConstraint     DiagnosticCode = "constraint"      // enum/min/max violation
	DiagScope          DiagnosticCode = "scope"           // key not writable in this scope
	DiagSecret         DiagnosticCode = "secret"          // secret material stored inline
	DiagDuplicateKey   DiagnosticCode = "duplicate_key"   // same key twice in one object
	DiagLegacyKey      DiagnosticCode = "legacy_key"      // legacy spelling still in use
	DiagConflictingKey DiagnosticCode = "conflicting_key" // legacy and canonical both present
	DiagDeprecated     DiagnosticCode = "deprecated"      // key is deprecated
)

// Diagnostic is one validation finding.
//
// Key is the canonical key the finding is about; LegacyKey carries the spelling
// actually written in the document when the two differ, so a report can name
// what the user typed without losing the canonical identity. Fields are omitted
// rather than emitted empty: an absent field means "not applicable here", which
// an empty string cannot express.
type Diagnostic struct {
	Code        DiagnosticCode `json:"code"`
	Severity    Severity       `json:"severity"`
	File        string         `json:"file,omitempty"`
	Key         string         `json:"key,omitempty"`
	LegacyKey   string         `json:"legacy_key,omitempty"`
	Replacement string         `json:"replacement,omitempty"`
	Message     string         `json:"message"`
}

func (d Diagnostic) String() string {
	if d.ReportedKey() == "" {
		return d.Message
	}
	return d.ReportedKey() + ": " + d.Message
}

// ReportedKey is the spelling to show the user: the legacy form when the
// document used one, otherwise the canonical key. Reporters use this so a
// diagnostic names what was actually typed instead of a key the file
// does not contain.
func (d Diagnostic) ReportedKey() string {
	if d.LegacyKey != "" {
		return d.LegacyKey
	}
	return d.Key
}

// ValidateOptions tunes document validation.
type ValidateOptions struct {
	// Scope is the scope the file backs; used to check per-key writable scopes.
	// Leave empty to skip the scope check.
	Scope Scope
	// Strict promotes legacy-key and deprecation warnings to errors.
	Strict bool
}

// ValidationResult is the outcome for one file.
//
// Path and Scope identify what was validated; KeyCount is the number of leaf
// keys the document declared. Keys is retained as the serialized name so the
// existing JSON contract does not shift.
type ValidationResult struct {
	Path        string       `json:"path"`
	Scope       Scope        `json:"scope,omitempty"`
	KeyCount    int          `json:"keys"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}

// File is the path that was validated. Retained as a method so callers reading
// the older field name keep compiling against one obvious accessor.
func (r ValidationResult) File() string { return r.Path }

// Valid reports whether the result carries no error-severity diagnostics.
func (r ValidationResult) Valid() bool {
	for _, d := range r.Diagnostics {
		if d.Severity == SeverityError {
			return false
		}
	}
	return true
}

// Errors returns only the error-severity diagnostics.
func (r ValidationResult) Errors() []Diagnostic { return r.filter(SeverityError) }

// Warnings returns only the warning-severity diagnostics.
func (r ValidationResult) Warnings() []Diagnostic { return r.filter(SeverityWarning) }

// Err returns a typed ERR_M_CONFIG error for the first error-severity
// diagnostic, or nil when the document is valid. This is the bridge that lets
// configuration loading reject an invalid file using the same engine that backs
// `m config validate`, rather than a second copy of the rules.
//
// The subject carries the file and the offending key so the message locates the
// problem; secret values never reach it because diagnostics are built through
// the redaction boundary.
func (r ValidationResult) Err() error {
	for _, d := range r.Errors() {
		subject := r.Path
		if k := d.ReportedKey(); k != "" {
			subject = r.Path + ":" + k
		}
		return apperr.New(apperr.Config, "config.load", subject, d.Message)
	}
	return nil
}

func (r ValidationResult) filter(s Severity) []Diagnostic {
	var out []Diagnostic
	for _, d := range r.Diagnostics {
		if d.Severity == s {
			out = append(out, d)
		}
	}
	return out
}

// ValidateFile validates one config file on disk. A missing file is valid and
// yields no diagnostics, because an absent config is a legal state.
func ValidateFile(path string, opts ValidateOptions) ValidationResult {
	res := ValidationResult{Path: path, Scope: opts.Scope}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return res
		}
		res.Diagnostics = append(res.Diagnostics, Diagnostic{
			Code: DiagRead, Severity: SeverityError, File: path,
			Message: "cannot read file: " + err.Error(),
		})
		return res
	}
	return ValidateDocument(b, path, opts)
}

// ValidateDocument validates raw JSONC bytes. path is used only for reporting.
//
// Checks, in order: JSONC syntax, root shape, duplicate keys, then per-key
// existence, type, constraints, writable scope, secret-inlining, legacy
// spelling, and deprecation. Diagnostics are returned in a deterministic order
// so two runs over the same bytes produce byte-identical reports.
//
// The validator only reads: it never writes a file or mutates configuration.
func ValidateDocument(src []byte, path string, opts ValidateOptions) ValidationResult {
	res := ValidationResult{Path: path, Scope: opts.Scope}
	// canon is the canonical key; legacy is the spelling the document used when
	// it differs, so the report can name both without conflating them.
	add := func(code DiagnosticCode, sev Severity, canon, legacy, msg string) {
		res.Diagnostics = append(res.Diagnostics, Diagnostic{
			Code: code, Severity: sev, File: path,
			Key: canon, LegacyKey: legacy, Message: msg,
		})
	}

	parsed, err := ParseJSONC(src)
	if err != nil {
		add(DiagSyntax, SeverityError, "", "", "invalid JSONC: "+err.Error())
		return res
	}
	m, ok := parsed.(map[string]any)
	if !ok {
		add(DiagRoot, SeverityError, "", "", "root must be an object")
		return res
	}
	if dupErr := DetectDuplicateKeys(src); dupErr != nil {
		var dk *DuplicateKeyError
		key := ""
		if asDuplicate(dupErr, &dk) {
			key = dk.Path
		}
		add(DiagDuplicateKey, SeverityError, key, "",
			"duplicate key; the later value silently wins")
	}

	flat := flattenDotted(m, "")
	keys := make([]string, 0, len(flat))
	for k := range flat {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	res.KeyCount = len(keys)

	// Conflict detection works on flattened dotted paths, so a nested legacy
	// spelling ({"resolve":{"autoInstallPeers":true}}) is compared against the
	// nested canonical form rather than looked up as a literal dotted member of
	// the root map, which would never match.
	present := make(map[string]bool, len(keys))
	for _, k := range keys {
		present[k] = true
	}

	legacySeverity := SeverityWarning
	if opts.Strict {
		legacySeverity = SeverityError
	}

	for _, k := range keys {
		v := flat[k]
		canon, isLegacy, known := resolveKey(k)
		if !known {
			// Free-form namespaces stay legal; anything else in an owned
			// namespace is a typo the user wants to hear about.
			if err := validateUnknownKey(k, v); err != nil {
				add(DiagUnknownKey, SeverityError, k, "", err.Error())
				continue
			}
			// A free-form key has no spec, but registries.* still has a contract:
			// the value is a registry URL. Only that namespace is checked here;
			// other unowned keys are deliberately left alone, since treating
			// them as typed would turn every free-form key into an error.
			if strings.HasPrefix(k, "registries.") {
				if err := validateKeyValue(k, v); err != nil {
					add(DiagType, SeverityError, k, "", err.Error())
				}
			}
			continue
		}
		// legacy is empty for a canonical spelling, so a report only mentions a
		// legacy form when the document actually used one.
		legacy := ""
		if isLegacy {
			legacy = k
		}
		if isLegacy {
			if present[canon] {
				add(DiagConflictingKey, SeverityError, canon, legacy,
					fmt.Sprintf("conflicts with %q; remove one", canon))
			} else {
				d := Diagnostic{
					Code: DiagLegacyKey, Severity: legacySeverity, File: path,
					Key: canon, LegacyKey: legacy, Replacement: canon,
					Message: "use " + canon,
				}
				res.Diagnostics = append(res.Diagnostics, d)
			}
		}
		// Keys whose contract is "name of an environment variable" must not
		// hold the credential itself. Checked before validateKeyValue, which
		// applies the same rule but reports it as a plain type error and so
		// cannot distinguish a pasted secret from a mistyped value. Keys that
		// legitimately store a value (a public key, for instance) are Secret
		// only for redaction and are not checked here.
		if strings.HasSuffix(canon, "_env") {
			if s, isStr := v.(string); isStr {
				if envErr := validateEnvVarName(s); envErr != nil {
					add(DiagSecret, SeverityError, canon, legacy, envErr.Error())
					continue
				}
			}
		}
		if err := validateKeyValue(canon, v); err != nil {
			// A secret key's value must not reach the message; the schema-driven
			// redaction boundary decides, not the shape of the value.
			add(typeOrConstraintCode(canon, v), SeverityError, canon, legacy,
				redactDiagnosticMessage(canon, err.Error()))
			continue
		}
		spec := KeySpec(canon)
		if spec == nil {
			continue
		}
		if err := validateRange(spec, v); err != nil {
			add(DiagConstraint, SeverityError, canon, legacy, err.Error())
		}
		if opts.Scope != "" && opts.Scope != ScopeEffective && !scopeAllows(spec, opts.Scope) {
			add(DiagScope, SeverityError, canon, legacy,
				fmt.Sprintf("not writable in %s scope; allowed: %s",
					opts.Scope, strings.Join(scopeNames(spec), ", ")))
		}
		if spec.Deprecated {
			msg := "deprecated"
			if spec.Replacement != "" {
				msg += "; use " + spec.Replacement
			}
			d := Diagnostic{
				Code: DiagDeprecated, Severity: legacySeverity, File: path,
				Key: canon, LegacyKey: legacy, Replacement: spec.Replacement,
				Message: msg,
			}
			res.Diagnostics = append(res.Diagnostics, d)
		}
	}
	sortDiagnostics(res.Diagnostics)
	return res
}

// sortDiagnostics imposes a total order: file, then reported key, then code.
// Two validations of the same bytes therefore produce identical reports, which
// is what lets callers diff them and tests assert on them.
func sortDiagnostics(ds []Diagnostic) {
	sort.SliceStable(ds, func(i, j int) bool {
		a, b := ds[i], ds[j]
		if a.File != b.File {
			return a.File < b.File
		}
		if a.ReportedKey() != b.ReportedKey() {
			return a.ReportedKey() < b.ReportedKey()
		}
		return a.Code < b.Code
	})
}

// redactDiagnosticMessage strips a secret key's value out of a message built by
// a lower-level validator that has no notion of secrecy.
func redactDiagnosticMessage(key, msg string) string {
	if !IsSecret(key) {
		return msg
	}
	// The value is not recoverable from the message safely, so report the
	// constraint without it.
	return "invalid value for " + key + " (value withheld)"
}

// asDuplicate is a tiny errors.As shim kept local so the diagnostic path does
// not need the errors import in every branch.
func asDuplicate(err error, target **DuplicateKeyError) bool {
	dk, ok := err.(*DuplicateKeyError)
	if ok {
		*target = dk
	}
	return ok
}

// typeOrConstraintCode distinguishes a wrong Go type from a violated enum,
// so callers can tell "this is not a string" from "this is not an allowed value".
func typeOrConstraintCode(canon string, v any) DiagnosticCode {
	spec := KeySpec(canon)
	if spec == nil {
		return DiagType
	}
	if spec.Type == TypeEnum {
		if _, ok := v.(string); ok {
			return DiagConstraint
		}
	}
	if spec.Type == TypeDuration {
		if _, ok := v.(string); ok {
			return DiagConstraint
		}
	}
	return DiagType
}

// validateRange enforces Minimum/Maximum for integer keys.
func validateRange(spec *ConfigKeySpec, v any) error {
	if spec.Type != TypeInt {
		return nil
	}
	var n int64
	switch t := v.(type) {
	case int:
		n = int64(t)
	case int64:
		n = t
	case float64:
		n = int64(t)
	case json.Number:
		// ParseJSONC decodes with UseNumber, so an integer written in a config
		// file arrives here as json.Number. Without this case every range check
		// on a file-sourced value silently passed.
		parsed, err := t.Int64()
		if err != nil {
			return nil
		}
		n = parsed
	default:
		return nil
	}
	if spec.Minimum != nil && n < *spec.Minimum {
		return fmt.Errorf("%d is below minimum %d", n, *spec.Minimum)
	}
	if spec.Maximum != nil && n > *spec.Maximum {
		return fmt.Errorf("%d is above maximum %d", n, *spec.Maximum)
	}
	return nil
}

// scopeAllows reports whether spec may be written in scope. An empty Scopes
// list means every writable scope is allowed.
func scopeAllows(spec *ConfigKeySpec, scope Scope) bool {
	if len(spec.Scopes) == 0 {
		return true
	}
	for _, s := range spec.Scopes {
		if s == scope {
			return true
		}
	}
	return false
}

func scopeNames(spec *ConfigKeySpec) []string {
	if len(spec.Scopes) == 0 {
		return []string{string(ScopeUser), string(ScopeProject)}
	}
	out := make([]string, len(spec.Scopes))
	for i, s := range spec.Scopes {
		out[i] = string(s)
	}
	return out
}
