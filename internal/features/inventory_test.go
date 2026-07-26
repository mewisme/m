package features

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var indexedMVPs = []string{
	"0001", "0002", "0003", "0004", "0005", "0006", "0007", "0008", "0009",
	"0010", "0011", "0012", "0013", "0014", "0015", "0016", "0017", "0018", "0019",
	"0020", "0021", "0022", "0023", "0024", "0025", "0026", "0027", "0028", "0029",
	"0030", "0031",
	"0040", "0041", "0042", "0043", "0044", "0045", "0046",
	"0050", "0051", "0052", "0053", "0054", "0055", "0056", "0057",
	"0060", "0061", "0062",
	"0070", "0071", "0072", "0073", "0074",
	"0080", "0081", "0082", "0083", "0084", "0085", "0086", "0087", "0088", "0089",
	"0090",
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}

func TestValidateRejectsMissingPrimaryMVP(t *testing.T) {
	path := filepath.Join(repoRoot(t), "testdata", "features", "invalid-missing-mvp.json")
	_, err := LoadFile(path)
	if err == nil {
		t.Fatal("expected validation error for missing primary_mvp")
	}
}

func TestEveryIndexMVOOwnsFeature(t *testing.T) {
	inv, err := LoadEmbedded()
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateMVPCoverage(inv, indexedMVPs); err != nil {
		t.Fatal(err)
	}
}

func TestMewExtensionsMarkedExtension(t *testing.T) {
	inv, err := LoadEmbedded()
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateExtensions(inv); err != nil {
		t.Fatal(err)
	}
	mewOnly := []string{
		"foundation.features-inventory",
		"lockfile.m-lock",
		"runner.direct-shortcuts",
		"runner.interactive-select",
		"linker.reflink-planner",
		"exec.snapshot-capsule",
		"security.policy-as-code",
		"cross.future-backlog",
	}
	byID := make(map[string]Feature, len(inv.Features))
	for _, f := range inv.Features {
		byID[f.ID] = f
	}
	for _, id := range mewOnly {
		f, ok := byID[id]
		if !ok {
			t.Fatalf("missing Mew-only feature %q", id)
		}
		if f.CompatibilityClass != ClassExtension {
			t.Errorf("%q: want compatibility_class extension, got %q", id, f.CompatibilityClass)
		}
	}
}

func TestEmbeddedMatchesBaseline(t *testing.T) {
	embedded, err := LoadEmbedded()
	if err != nil {
		t.Fatal(err)
	}
	baseline := Baseline()
	if len(embedded.Features) != len(baseline.Features) {
		t.Fatalf("feature count: embedded=%d baseline=%d", len(embedded.Features), len(baseline.Features))
	}
	if err := ValidateMVPCoverage(embedded, indexedMVPs); err != nil {
		t.Fatal(err)
	}
}

func TestFormatJSONDeterministic(t *testing.T) {
	inv, err := LoadEmbedded()
	if err != nil {
		t.Fatal(err)
	}
	features := inv.Filter("", "")
	out1, err := FormatJSON(features)
	if err != nil {
		t.Fatal(err)
	}
	out2, err := FormatJSON(features)
	if err != nil {
		t.Fatal(err)
	}
	if string(out1) != string(out2) {
		t.Fatal("JSON output is not deterministic")
	}
	var decoded []PublicFeature
	if err := json.Unmarshal(out1, &decoded); err != nil {
		t.Fatalf("CLI JSON invalid: %v", err)
	}
	if len(decoded) != len(features) {
		t.Fatalf("decoded %d features, want %d", len(decoded), len(features))
	}
	for _, f := range decoded {
		if f.ID == "" || f.PrimaryMVP == "" {
			t.Fatalf("public feature missing required fields: %+v", f)
		}
	}
}

func TestFormatTableGolden(t *testing.T) {
	path := filepath.Join(repoRoot(t), "testdata", "features", "minimal-inventory.json")
	inv, err := LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := FormatTable(inv.Features)
	goldenPath := filepath.Join(repoRoot(t), "testdata", "features", "golden-table.txt")
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatal(err)
	}
	if normalizeEOL(got) != normalizeEOL(string(want)) {
		t.Fatalf("table mismatch:\n--- got ---\n%s--- want ---\n%s", got, want)
	}
}

func normalizeEOL(s string) string {
	return strings.ReplaceAll(s, "\r\n", "\n")
}

func TestWriteInventoryJSON(t *testing.T) {
	if os.Getenv("UPDATE_INVENTORY") != "1" {
		t.Skip("set UPDATE_INVENTORY=1 to regenerate features/inventory.json")
	}
	inv := Baseline()
	if err := Validate(inv); err != nil {
		t.Fatal(err)
	}
	if err := ValidateMVPCoverage(inv, indexedMVPs); err != nil {
		t.Fatal(err)
	}
	b, err := json.MarshalIndent(inv, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	b = append(b, '\n')
	out := filepath.Join(repoRoot(t), "features", "inventory.json")
	if err := os.WriteFile(out, b, 0o644); err != nil {
		t.Fatal(err)
	}
}
