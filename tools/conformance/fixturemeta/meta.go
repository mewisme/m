// Package fixturemeta defines the lock bridge fixture provenance schema and validation.
package fixturemeta

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Classification values for fixture metadata.
const (
	ClassGenerated = "generated"
	ClassDerived   = "derived"
)

// Meta is committed provenance for a lock bridge conformance fixture family.
type Meta struct {
	Producer        string   `json:"producer"`
	ProducerVersion string   `json:"producerVersion"`
	ProducerMajor   int      `json:"producerMajor"`
	Family          string   `json:"family"`
	Node            string   `json:"node"`
	OS              string   `json:"os"`
	Arch            string   `json:"arch"`
	ExecutablePath  string   `json:"executablePath"`
	ExecutableArgs  []string `json:"executableArgs"`
	Registry        string   `json:"registry"`
	Timestamp       string   `json:"timestamp"`
	Classification  string   `json:"classification"`
	LockfileVersion string   `json:"lockfileVersion"`
	LockfileSha256  string   `json:"lockfileSha256"`

	PackageJSONSha256       string            `json:"packageJsonSha256"`
	WorkspaceYAMLSha256     string            `json:"workspaceYamlSha256,omitempty"`
	WorkspaceManifestSha256 map[string]string `json:"workspaceManifestSha256,omitempty"`
	PatchFileSha256         map[string]string `json:"patchFileSha256,omitempty"`
	SourceTreeDigest        string            `json:"sourceTreeDigest"`
	InvocationID            string            `json:"invocationId"`
	IsolatedHomePolicy      string            `json:"isolatedHomePolicy"`

	// Derived Nub fixtures.
	SourceFixture     string `json:"sourceFixture,omitempty"`
	SourceLockSha256  string `json:"sourceLockSha256,omitempty"`
	DerivationCommand string `json:"derivationCommand,omitempty"`

	// Human-readable command summary (legacy field retained for logs and tests).
	Command string `json:"command"`

	Confidence        string   `json:"confidence,omitempty"`
	GenerationSignals []string `json:"generationSignals,omitempty"`
}

// ReadMeta decodes metadata.json from path.
func ReadMeta(path string) (Meta, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Meta{}, err
	}
	var meta Meta
	if err := json.Unmarshal(raw, &meta); err != nil {
		return Meta{}, err
	}
	return meta, nil
}

// Validate checks required fields and rejects placeholder or stale provenance.
func (m Meta) Validate(opts ValidateOptions) []string {
	var errs []string
	add := func(format string, args ...any) {
		errs = append(errs, fmt.Sprintf(format, args...))
	}
	if m.Producer == "" {
		add("missing producer")
	}
	if m.ProducerVersion == "" {
		add("missing producerVersion")
	}
	if m.ProducerMajor == 0 {
		add("missing producerMajor")
	}
	if m.Family == "" {
		add("missing family")
	}
	if m.Node == "" {
		add("missing node")
	}
	if m.OS == "" {
		add("missing os")
	}
	if m.Arch == "" {
		add("missing arch")
	}
	if m.ExecutablePath == "" {
		add("missing executablePath")
	}
	if len(m.ExecutableArgs) == 0 {
		add("missing executableArgs")
	}
	if m.Registry == "" {
		add("missing registry")
	}
	if m.Timestamp == "" {
		add("missing timestamp")
	}
	if m.Classification == "" {
		add("missing classification")
	}
	if m.LockfileVersion == "" {
		add("missing lockfileVersion")
	}
	if m.LockfileSha256 == "" {
		add("missing lockfileSha256")
	}
	if m.PackageJSONSha256 == "" {
		add("missing packageJsonSha256")
	}
	if m.SourceTreeDigest == "" {
		add("missing sourceTreeDigest")
	}
	if m.InvocationID == "" {
		add("missing invocationId")
	}
	if m.IsolatedHomePolicy == "" {
		add("missing isolatedHomePolicy")
	}
	if m.Command == "" {
		add("missing command")
	}
	if IsPlaceholderCommand(m.Command) {
		add("placeholder command %q", m.Command)
	}
	if opts.WantProducerVersion != "" && m.ProducerVersion != opts.WantProducerVersion {
		add("producerVersion=%q want %q", m.ProducerVersion, opts.WantProducerVersion)
	}
	if opts.WantProducerMajor != 0 && m.ProducerMajor != opts.WantProducerMajor {
		add("producerMajor=%d want %d", m.ProducerMajor, opts.WantProducerMajor)
	}
	if opts.WantFamily != "" && m.Family != opts.WantFamily {
		add("family=%q want %q", m.Family, opts.WantFamily)
	}
	if opts.WantProducer != "" && m.Producer != opts.WantProducer {
		add("producer=%q want %q", m.Producer, opts.WantProducer)
	}
	if m.Classification == ClassDerived {
		if m.SourceFixture == "" {
			add("derived fixture missing sourceFixture")
		}
		if m.SourceLockSha256 == "" {
			add("derived fixture missing sourceLockSha256")
		}
		if m.DerivationCommand == "" {
			add("derived fixture missing derivationCommand")
		}
	}
	return errs
}

// ValidateOptions scopes metadata validation for a fixture tree entry.
type ValidateOptions struct {
	WantProducer        string
	WantProducerVersion string
	WantProducerMajor   int
	WantFamily          string
}

// IsPlaceholderCommand reports invented or stale command strings.
func IsPlaceholderCommand(cmd string) bool {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return false
	}
	placeholders := []string{
		"committed generated fixture",
		"placeholder",
		"todo",
	}
	lower := strings.ToLower(cmd)
	for _, p := range placeholders {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}

// VerifyFixtureDir re-reads files under dir and checks metadata against on-disk bytes.
func VerifyFixtureDir(dir string, meta Meta, lockName string) []string {
	var errs []string
	add := func(format string, args ...any) {
		errs = append(errs, fmt.Sprintf(format, args...))
	}
	lockPath := filepath.Join(dir, lockName)
	lockData, err := os.ReadFile(lockPath)
	if err != nil {
		add("lock: %v", err)
		return errs
	}
	if got := FileSHA256Bytes(lockData); meta.LockfileSha256 != got {
		add("lockfileSha256 mismatch meta=%s calc=%s", meta.LockfileSha256, got)
	}
	pkgPath := filepath.Join(dir, "package.json")
	pkgData, err := os.ReadFile(pkgPath)
	if err != nil {
		add("package.json: %v", err)
	} else if got := FileSHA256Bytes(pkgData); meta.PackageJSONSha256 != got {
		add("packageJsonSha256 mismatch meta=%s calc=%s", meta.PackageJSONSha256, got)
	}
	wsPath := filepath.Join(dir, "pnpm-workspace.yaml")
	if wsData, err := os.ReadFile(wsPath); err == nil {
		if meta.WorkspaceYAMLSha256 == "" {
			add("missing workspaceYamlSha256 for present pnpm-workspace.yaml")
		} else if got := FileSHA256Bytes(wsData); meta.WorkspaceYAMLSha256 != got {
			add("workspaceYamlSha256 mismatch meta=%s calc=%s", meta.WorkspaceYAMLSha256, got)
		}
	}
	for rel, want := range meta.WorkspaceManifestSha256 {
		path := filepath.Join(dir, filepath.FromSlash(rel))
		data, err := os.ReadFile(path)
		if err != nil {
			add("workspace manifest %s: %v", rel, err)
			continue
		}
		if got := FileSHA256Bytes(data); got != want {
			add("workspace manifest %s sha mismatch meta=%s calc=%s", rel, want, got)
		}
	}
	for rel, want := range meta.PatchFileSha256 {
		path := filepath.Join(dir, filepath.FromSlash(rel))
		data, err := os.ReadFile(path)
		if err != nil {
			add("patch file %s: %v", rel, err)
			continue
		}
		if got := FileSHA256Bytes(data); got != want {
			add("patch file %s sha mismatch meta=%s calc=%s", rel, want, got)
		}
	}
	return errs
}
