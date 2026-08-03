package config

import (
	"fmt"
	"os"
	"sort"
	"strings"
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
type Diagnostic struct {
	Code     DiagnosticCode `json:"code"`
	Severity Severity       `json:"severity"`
	File     string         `json:"file,omitempty"`
	Key      string         `json:"key,omitempty"`
	Message  string         `json:"message"`
}

func (d Diagnostic) String() string {
	if d.Key == "" {
		return d.Message
	}
	return d.Key + ": " + d.Message
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
type ValidationResult struct {
	File        string       `json:"file"`
	Keys        int          `json:"keys"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}

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
	res := ValidationResult{File: path}
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
// spelling, and deprecation.
func ValidateDocument(src []byte, path string, opts ValidateOptions) ValidationResult {
	res := ValidationResult{File: path}
	add := func(code DiagnosticCode, sev Severity, key, msg string) {
		res.Diagnostics = append(res.Diagnostics, Diagnostic{
			Code: code, Severity: sev, File: path, Key: key, Message: msg,
		})
	}

	parsed, err := ParseJSONC(src)
	if err != nil {
		add(DiagSyntax, SeverityError, "", "invalid JSONC: "+err.Error())
		return res
	}
	m, ok := parsed.(map[string]any)
	if !ok {
		add(DiagRoot, SeverityError, "", "root must be an object")
		return res
	}
	if dupErr := DetectDuplicateKeys(src); dupErr != nil {
		var dk *DuplicateKeyError
		key := ""
		if asDuplicate(dupErr, &dk) {
			key = dk.Path
		}
		add(DiagDuplicateKey, SeverityError, key,
			"duplicate key; the later value silently wins")
	}

	flat := flattenDotted(m, "")
	keys := make([]string, 0, len(flat))
	for k := range flat {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	res.Keys = len(keys)

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
				add(DiagUnknownKey, SeverityError, k, err.Error())
			}
			continue
		}
		if isLegacy {
			if present[canon] {
				add(DiagConflictingKey, SeverityError, k,
					fmt.Sprintf("conflicts with %q; remove one", canon))
			} else {
				add(DiagLegacyKey, legacySeverity, k, "use "+canon)
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
					add(DiagSecret, SeverityError, k, envErr.Error())
					continue
				}
			}
		}
		if err := validateKeyValue(canon, v); err != nil {
			add(typeOrConstraintCode(canon, v), SeverityError, k, err.Error())
			continue
		}
		spec := KeySpec(canon)
		if spec == nil {
			continue
		}
		if err := validateRange(spec, v); err != nil {
			add(DiagConstraint, SeverityError, k, err.Error())
		}
		if opts.Scope != "" && opts.Scope != ScopeEffective && !scopeAllows(spec, opts.Scope) {
			add(DiagScope, SeverityError, k,
				fmt.Sprintf("not writable in %s scope; allowed: %s",
					opts.Scope, strings.Join(scopeNames(spec), ", ")))
		}
		if spec.Deprecated {
			msg := "deprecated"
			if spec.Replacement != "" {
				msg += "; use " + spec.Replacement
			}
			add(DiagDeprecated, legacySeverity, k, msg)
		}
	}
	return res
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
