package pnpm

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/lockfile"
)

func TestFieldLossAuditSettings(t *testing.T) {
	doc := &Document{
		LockfileVersion: "9.0",
		Settings:        map[string]any{"autoInstallPeers": true},
		Importers:       map[string]ImporterSection{".": {}},
		Packages:        map[string]PackageEntry{},
		Snapshots:       map[string]map[string]any{},
	}
	report := FieldLossAudit(doc)
	if !hasSemanticLoss(report, "settings.autoInstallPeers") {
		t.Fatalf("items=%v", report.Items)
	}
}

func TestFieldLossAuditImporterMeta(t *testing.T) {
	doc := &Document{
		LockfileVersion: "9.0",
		Importers: map[string]ImporterSection{
			".": {
				DependenciesMeta: map[string]any{"pkg": map[string]any{"injected": true}},
			},
		},
		Packages:  map[string]PackageEntry{},
		Snapshots: map[string]map[string]any{},
	}
	report := FieldLossAudit(doc)
	if len(SemanticLossItems(report)) == 0 {
		t.Fatal("expected semantic loss for importer metadata")
	}
}

func TestFieldLossAuditEngines(t *testing.T) {
	doc := &Document{
		LockfileVersion: "9.0",
		Importers:       map[string]ImporterSection{".": {}},
		Packages: map[string]PackageEntry{
			"pkg@1.0.0": {Engines: map[string]any{"node": ">=18"}},
		},
		Snapshots: map[string]map[string]any{},
	}
	report := FieldLossAudit(doc)
	if !hasSemanticLoss(report, "packages.pkg@1.0.0.engines") {
		t.Fatalf("items=%v", report.Items)
	}
}

func TestFieldLossAuditTarballResolution(t *testing.T) {
	doc := &Document{
		LockfileVersion: "9.0",
		Importers:       map[string]ImporterSection{".": {}},
		Packages: map[string]PackageEntry{
			"pkg@1.0.0": {Resolution: map[string]any{"tarball": "https://example.com/pkg.tgz"}},
		},
		Snapshots: map[string]map[string]any{},
	}
	report := FieldLossAudit(doc)
	if !hasSemanticLoss(report, "packages.pkg@1.0.0.resolution.tarball") {
		t.Fatalf("items=%v", report.Items)
	}
}

func TestFieldLossAuditGitResolution(t *testing.T) {
	doc := &Document{
		LockfileVersion: "9.0",
		Importers:       map[string]ImporterSection{".": {}},
		Packages: map[string]PackageEntry{
			"pkg@1.0.0": {Resolution: map[string]any{"repo": "github.com/o/r", "type": "git"}},
		},
		Snapshots: map[string]map[string]any{},
	}
	report := FieldLossAudit(doc)
	if !hasSemanticLossPrefix(report, "packages.pkg@1.0.0.resolution.") {
		t.Fatalf("items=%v", report.Items)
	}
}

func TestFieldLossAuditDirectoryResolution(t *testing.T) {
	doc := &Document{
		LockfileVersion: "9.0",
		Importers:       map[string]ImporterSection{".": {}},
		Packages: map[string]PackageEntry{
			"pkg@1.0.0": {Resolution: map[string]any{"directory": "../local", "type": "directory"}},
		},
		Snapshots: map[string]map[string]any{},
	}
	report := FieldLossAudit(doc)
	if !hasSemanticLoss(report, "packages.pkg@1.0.0.resolution.directory") {
		t.Fatalf("items=%v", report.Items)
	}
}

func TestFieldLossAuditPlatformConstraints(t *testing.T) {
	doc := &Document{
		LockfileVersion: "9.0",
		Importers:       map[string]ImporterSection{".": {}},
		Packages: map[string]PackageEntry{
			"pkg@1.0.0": {Extra: map[string]any{"os": "linux", "cpu": "x64", "libc": "glibc"}},
		},
		Snapshots: map[string]map[string]any{},
	}
	report := FieldLossAudit(doc)
	for _, field := range []string{"os", "cpu", "libc"} {
		want := "packages.pkg@1.0.0." + field
		if !hasSemanticLoss(report, want) {
			t.Fatalf("missing %s in %v", want, report.Items)
		}
	}
}

func TestFieldLossAuditTransitivePeerDeps(t *testing.T) {
	doc := &Document{
		LockfileVersion: "9.0",
		Importers:       map[string]ImporterSection{".": {}},
		Packages:        map[string]PackageEntry{},
		Snapshots: map[string]map[string]any{
			"pkg@1.0.0": {"transitivePeerDependencies": []any{"peer@1.0.0"}},
		},
	}
	report := FieldLossAudit(doc)
	if !hasSemanticLoss(report, "snapshots.pkg@1.0.0.transitivePeerDependencies") {
		t.Fatalf("items=%v", report.Items)
	}
}

func TestFieldLossAuditRegistryIntegrityMapped(t *testing.T) {
	doc := &Document{
		LockfileVersion: "9.0",
		Importers:       map[string]ImporterSection{".": {}},
		Packages: map[string]PackageEntry{
			"pkg@1.0.0": {Resolution: map[string]any{"integrity": "sha512-test"}},
		},
		Snapshots: map[string]map[string]any{},
	}
	report := FieldLossAudit(doc)
	for _, item := range report.Items {
		if strings.Contains(item.Field, "resolution") {
			t.Fatalf("registry integrity should map, got %v", item)
		}
	}
}

func TestFieldLossAuditTopLevelCatalogsOverrides(t *testing.T) {
	doc := &Document{
		LockfileVersion: "9.0",
		Importers:       map[string]ImporterSection{".": {}},
		Packages:        map[string]PackageEntry{},
		Snapshots:       map[string]map[string]any{},
		Extensions: lockfile.Extensions{
			"catalogs":  json.RawMessage(`{}`),
			"overrides": json.RawMessage(`{}`),
			"time":      json.RawMessage(`"2024-01-01"`),
		},
	}
	report := FieldLossAudit(doc)
	for _, field := range []string{"catalogs", "overrides", "time"} {
		if !hasSemanticLoss(report, field) {
			t.Fatalf("missing %s in %v", field, report.Items)
		}
	}
}

func hasSemanticLoss(report lockfile.LossReport, field string) bool {
	for _, item := range SemanticLossItems(report) {
		if item.Field == field {
			return true
		}
	}
	return false
}

func hasSemanticLossPrefix(report lockfile.LossReport, prefix string) bool {
	for _, item := range SemanticLossItems(report) {
		if strings.HasPrefix(item.Field, prefix) {
			return true
		}
	}
	return false
}
