package conformance_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mewisme/m/internal/testkit"
)

func TestConformanceInventoryStub(t *testing.T) {
	root := testkit.ModuleRoot(t)
	path := filepath.Join(root, "tests", "conformance", "inventory.json")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		SchemaVersion int `json:"schemaVersion"`
		Cases         []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatal(err)
	}
	if doc.SchemaVersion != 1 || len(doc.Cases) == 0 {
		t.Fatalf("bad inventory: %+v", doc)
	}
}

func TestDifferentialSmoke(t *testing.T) {
	home := testkit.TempHome(t)
	reportPath := filepath.Join(home, "diff-report.json")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	report := &testkit.DiffReport{
		SchemaVersion: testkit.DiffReportSchemaVersion,
		Diffs:         []testkit.DiffItem{},
	}

	npmPath := testkit.LookPM("npm")
	nubPath := testkit.LookPM("nub")
	refName := "npm"
	refPath := npmPath
	if refPath == "" {
		refName = "nub"
		refPath = nubPath
	}

	// Mew side: always available via go run version (cheap process).
	mew := testkit.RunPM(ctx, "go", []string{"run", "./cmd/m", "version"}, testkit.ModuleRoot(t), os.Environ())
	report.Mew = testkit.ToolRun{
		Name:     "m",
		Args:     []string{"version"},
		ExitCode: mew.ExitCode,
		Stdout:   testkit.NormalizeOutput(mew.Stdout),
		Stderr:   testkit.NormalizeOutput(mew.Stderr),
	}

	if refPath == "" {
		report.Skipped = true
		report.SkipReason = "reference PM (npm/nub) not found on PATH"
		report.Reference = testkit.ToolRun{Name: "npm", ExitCode: 127}
	} else {
		ref := testkit.RunPM(ctx, refName, []string{"--version"}, home, os.Environ())
		report.Reference = testkit.ToolRun{
			Name:     refName,
			Args:     []string{"--version"},
			ExitCode: ref.ExitCode,
			Stdout:   testkit.NormalizeOutput(ref.Stdout),
			Stderr:   testkit.NormalizeOutput(ref.Stderr),
		}
		if report.Mew.ExitCode != 0 {
			report.Diffs = append(report.Diffs, testkit.DiffItem{
				Field:  "mew.exitCode",
				Mew:    "nonzero",
				Detail: "m version failed",
			})
		}
	}

	if err := testkit.WriteDiffReport(reportPath, report); err != nil {
		t.Fatal(err)
	}
	got, err := testkit.ReadDiffReport(reportPath)
	if err != nil {
		t.Fatal(err)
	}
	if got.SchemaVersion != testkit.DiffReportSchemaVersion {
		t.Fatalf("schema %d", got.SchemaVersion)
	}
	if report.Skipped && !got.Skipped {
		t.Fatal("expected skipped report")
	}
}
