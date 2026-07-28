package lifecycle

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/mewisme/mew/internal/apperr"
)

// AuditEntry is one lifecycle script execution record.
type AuditEntry struct {
	TS           string                 `json:"ts"`
	Package      string                 `json:"package"`
	Script       string                 `json:"script"`
	ExitCode     int                    `json:"exitCode"`
	DurationMs   int64                  `json:"durationMs"`
	Cached       bool                   `json:"cached,omitempty"`
	Restored     bool                   `json:"restored,omitempty"`
	TimedOut     bool                   `json:"timedOut,omitempty"`
	Status       string                 `json:"status,omitempty"`
	Capabilities *ExecutionCapabilities `json:"capabilities,omitempty"`
}

// AppendAudit appends one redacted audit record.
func AppendAudit(path string, e AuditEntry) error {
	if e.TS == "" {
		e.TS = time.Now().UTC().Format(time.RFC3339Nano)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return apperr.Wrap(apperr.IO, "lifecycle.audit", path, err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return apperr.Wrap(apperr.IO, "lifecycle.audit", path, err)
	}
	defer func() { _ = f.Close() }()
	enc := json.NewEncoder(f)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(e); err != nil {
		return apperr.Wrap(apperr.IO, "lifecycle.audit", path, err)
	}
	return nil
}

// ReadAudit loads all audit entries from path.
func ReadAudit(path string) ([]AuditEntry, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, apperr.Wrap(apperr.IO, "lifecycle.audit", path, err)
	}
	var out []AuditEntry
	for _, line := range splitLines(string(raw)) {
		line = trimSpace(line)
		if line == "" {
			continue
		}
		var e AuditEntry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			return nil, apperr.Wrap(apperr.IO, "lifecycle.audit", path, err)
		}
		out = append(out, e)
	}
	return out, nil
}

func splitLines(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}

func trimSpace(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t' || s[0] == '\r') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t' || s[len(s)-1] == '\r') {
		s = s[:len(s)-1]
	}
	return s
}
