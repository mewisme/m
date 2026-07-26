package releasetrain_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/mewisme/m/internal/features"
	"github.com/mewisme/m/internal/releasetrain"
)

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

func loadGraph(t *testing.T) *releasetrain.Graph {
	t.Helper()
	g, err := releasetrain.LoadFile(filepath.Join(repoRoot(t), "features", "milestones.json"))
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func TestMilestonesAcyclicAndValid(t *testing.T) {
	g := loadGraph(t)
	if err := g.ValidateStabilizationOrder(); err != nil {
		t.Fatal(err)
	}
	if err := g.ValidateNonBlocking0090(); err != nil {
		t.Fatal(err)
	}
}

func TestIndexMatchesMilestones(t *testing.T) {
	g := loadGraph(t)
	indexPath := filepath.Join(repoRoot(t), "plans", "INDEX.md")
	b, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatal(err)
	}
	re := regexp.MustCompile("`([0-9]{4})-")
	found := make(map[string]struct{})
	for _, m := range re.FindAllStringSubmatch(string(b), -1) {
		id := m[1]
		if id == "0000" {
			continue // archive overview, not a delivery milestone
		}
		found[id] = struct{}{}
	}
	for _, id := range g.IDs() {
		if _, ok := found[id]; !ok {
			t.Errorf("INDEX.md missing milestone %s", id)
		}
	}
	for id := range found {
		if _, ok := g.ByID(id); !ok {
			t.Errorf("INDEX.md has %s not in milestones.json", id)
		}
	}
	if !strings.Contains(string(b), "docs/release-train.md") && !strings.Contains(string(b), "../docs/release-train.md") {
		// Allow either relative form from plans/
		if !strings.Contains(string(b), "release-train.md") {
			t.Error("INDEX.md must link release-train.md")
		}
	}
}

func TestInventoryPrimaryMVPInMilestones(t *testing.T) {
	g := loadGraph(t)
	inv, err := features.LoadEmbedded()
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range inv.Features {
		if _, ok := g.ByID(f.PrimaryMVP); !ok {
			t.Errorf("feature %s primary_mvp %s missing from milestones", f.ID, f.PrimaryMVP)
		}
	}
}

func TestReleaseTrainDocStopTheLine(t *testing.T) {
	path := filepath.Join(repoRoot(t), "docs", "release-train.md")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := strings.ToLower(string(b))
	for _, needle := range []string{"integrity", "corruption", "credential"} {
		if !strings.Contains(text, needle) {
			t.Errorf("release-train.md missing stop-the-line term %q", needle)
		}
	}
}

func TestEmptyScaffoldChecklist(t *testing.T) {
	path := filepath.Join(repoRoot(t), "testdata", "release", "empty-scaffold-checklist.md")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(b)
	for _, h := range []string{"## Preconditions", "## Build and test", "## Hermetic and experimental policy", "## Stop-the-line"} {
		if !strings.Contains(text, h) {
			t.Errorf("checklist missing heading %q", h)
		}
	}
}
