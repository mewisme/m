package pnpm

import "testing"

func TestFieldLossAuditSettings(t *testing.T) {
	doc := &Document{
		LockfileVersion: "9.0",
		Settings:        map[string]any{"autoInstallPeers": true},
		Importers:       map[string]ImporterSection{".": {}},
		Packages:        map[string]PackageEntry{},
		Snapshots:       map[string]map[string]any{},
	}
	report := FieldLossAudit(doc)
	if len(report.Items) == 0 {
		t.Fatal("expected settings loss")
	}
	found := false
	for _, item := range report.Items {
		if item.Field == "settings.autoInstallPeers" && item.Semantic {
			found = true
		}
	}
	if !found {
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
