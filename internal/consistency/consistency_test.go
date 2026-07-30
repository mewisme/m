package consistency_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/features"
)

type planStatus struct {
	CompletedMvps       []string `json:"completedMvps"`
	InventoryExceptions []string `json:"inventoryExceptions"`
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

func loadPlanStatus(t *testing.T, root string) planStatus {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, "plans", "status.json"))
	if err != nil {
		t.Fatal(err)
	}
	var status planStatus
	if err := json.Unmarshal(b, &status); err != nil {
		t.Fatal(err)
	}
	return status
}

func TestCompletedMvpsOwnShippedInventoryRows(t *testing.T) {
	root := repoRoot(t)
	status := loadPlanStatus(t, root)
	exceptions := map[string]struct{}{}
	for _, id := range status.InventoryExceptions {
		exceptions[id] = struct{}{}
	}
	completed := map[string]struct{}{}
	for _, id := range status.CompletedMvps {
		completed[id] = struct{}{}
	}

	inv, err := features.LoadFile(filepath.Join(root, "features", "inventory.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range inv.Features {
		if f.Module != "foundation" {
			continue
		}
		if _, ok := completed[f.PrimaryMVP]; !ok {
			continue
		}
		if _, ok := exceptions[f.ID]; ok {
			continue
		}
		if f.MewStatus == features.StatusInProgress || f.MewStatus == features.StatusPlanned {
			t.Errorf("completed MVP %s still owns inventory row %s with mew_status=%s", f.PrimaryMVP, f.ID, f.MewStatus)
		}
	}
}

func TestPackageMapCertifiedPathsHaveInventoryEvidence(t *testing.T) {
	root := repoRoot(t)
	mapText, err := os.ReadFile(filepath.Join(root, "docs", "architecture", "package-map.md"))
	if err != nil {
		t.Fatal(err)
	}
	inv, err := features.LoadFile(filepath.Join(root, "features", "inventory.json"))
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]features.Feature{}
	for _, f := range inv.Features {
		byID[f.ID] = f
	}

	cases := []struct {
		path       string
		capability string
		featureID  string
	}{
		{"internal/sbom/", "certified", "security.sbom"},
		{"internal/provenance/", "shipped", "security.provenance"},
		{"internal/features/", "shipped", "foundation.features-inventory"},
	}
	rowRE := regexp.MustCompile("`([^`]+)`.*\\|\\s*(exists|reserved|absent)\\s*\\|\\s*(\\S+)\\s*\\|")
	for _, line := range strings.Split(string(mapText), "\n") {
		if !strings.Contains(line, "|") || strings.Contains(line, "Path |") {
			continue
		}
		m := rowRE.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		for _, c := range cases {
			if m[1] != c.path && m[1] != strings.TrimSuffix(c.path, "/") {
				continue
			}
			if m[3] != c.capability {
				t.Errorf("package-map %s capability=%s want %s", c.path, m[3], c.capability)
			}
			f := byID[c.featureID]
			if f.MewStatus != features.StatusShipped {
				t.Errorf("feature %s mew_status=%s want shipped for package-map %s", c.featureID, f.MewStatus, c.path)
			}
			if len(f.Tests) == 0 {
				t.Errorf("feature %s missing tests for shipped package-map path %s", c.featureID, c.path)
			}
		}
	}
}

func TestREADMECertifiedClaimMatchesInventory(t *testing.T) {
	root := repoRoot(t)
	readme, err := os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(readme), "**Certified**") {
		t.Fatal("README missing certified PM core claim")
	}
	inv, err := features.LoadFile(filepath.Join(root, "features", "inventory.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range inv.Features {
		if f.ID != "foundation.core-stabilization" {
			continue
		}
		if f.MewStatus != features.StatusShipped {
			t.Fatalf("foundation.core-stabilization mew_status=%s", f.MewStatus)
		}
	}
}
