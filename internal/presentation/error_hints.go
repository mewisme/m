package presentation

import (
	"strings"

	"github.com/mewisme/mew/internal/apperr"
)

const maxDefaultHints = 3

// ErrorMetadata carries typed fields for hint predicates.
type ErrorMetadata struct {
	Code      apperr.Code
	Operation string
	Subject   string
}

type hintRule struct {
	code      apperr.Code
	operation string
	predicate func(ErrorMetadata) bool
	message   string
}

var hintRules = []hintRule{
	{code: apperr.Usage, message: "Run `m <command> --help` for usage."},
	{code: apperr.Lockfile, operation: "install", message: "Run `m install` to update the lockfile."},
	{code: apperr.Lockfile, message: "Run `m install` to refresh the lockfile."},
	{code: apperr.NotFound, predicate: func(m ErrorMetadata) bool {
		return strings.Contains(m.Operation, "run") || strings.Contains(m.Subject, "script")
	}, message: "Run `m run <script>` or list scripts with `m pkg get scripts`."},
	{code: apperr.NotFound, message: "Check the project path and package name."},
	{code: apperr.Policy, operation: "lifecycle.trust", message: "Review and approve with `m trust <package>` or `m builds`."},
	{code: apperr.Policy, message: "Review policy with `m policy check` or lifecycle trust with `m builds`."},
	{code: apperr.Config, message: "Run `m config list` to inspect effective configuration."},
	{code: apperr.Resolve, message: "Run `m explain <package>` to inspect resolution decisions."},
	{code: apperr.Transaction, message: "Run `m recover` to inspect incomplete transactions."},
	{code: apperr.Integrity, operation: "doctor", message: "Run `m doctor` for a health report."},
	{code: apperr.Integrity, message: "Run `m verify` or `m doctor` to inspect integrity state."},
	{code: apperr.Network, message: "Check network connectivity or use `--offline` when cache is warm."},
	{code: apperr.Manifest, message: "Validate package.json syntax and required fields."},
	{code: apperr.Store, message: "Run `m store status` to inspect the content store."},
	{code: apperr.Unsupported, message: "See `m features` for supported capabilities."},
	{code: apperr.Unimplemented, message: "See `m features` for planned commands."},
	{code: apperr.Exec, message: "Run `m exec <binary> --help` for launch options."},
	{code: apperr.Timeout, message: "Retry the operation or increase configured timeouts."},
	{code: apperr.IO, message: "Check permissions and disk space for the affected path."},
}

// HintsFor returns deterministic hints for an error (capped in default mode).
func HintsFor(meta ErrorMetadata, debug bool) []Hint {
	if meta.Code == apperr.Cancelled {
		return nil
	}
	var out []Hint
	seen := make(map[string]struct{})
	for _, rule := range hintRules {
		if rule.code != "" && rule.code != meta.Code {
			continue
		}
		if rule.operation != "" && !strings.Contains(meta.Operation, rule.operation) {
			continue
		}
		if rule.predicate != nil && !rule.predicate(meta) {
			continue
		}
		if _, ok := seen[rule.message]; ok {
			continue
		}
		seen[rule.message] = struct{}{}
		out = append(out, Hint{Message: rule.message})
		if !debug && len(out) >= maxDefaultHints {
			break
		}
	}
	return out
}
