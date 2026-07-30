package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestBenchMedianP95(t *testing.T) {
	t.Parallel()
	samples := []int64{10, 20, 30, 40, 100}
	if got := benchMedian(samples); got != 30 {
		t.Fatalf("median=%d", got)
	}
	if got := benchP95(samples); got != 100 {
		t.Fatalf("p95=%d", got)
	}
	if got := benchMedian([]int64{10, 20}); got != 15 {
		t.Fatalf("even median=%d", got)
	}
}

func TestUpdateInstallBaselineRoundTrip(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "benchmarks"), 0o755); err != nil {
		t.Fatal(err)
	}
	result := BenchResult{
		Case:          "medium-graph-warm",
		Mode:          "warm",
		Samples:       []int64{100, 110, 105},
		MedianMs:      105,
		P95Ms:         110,
		TotalMs:       105,
		GoVersion:     "go1.26.0",
		OS:            "linux",
		Arch:          "amd64",
		RunnerClass:   "github-actions-ubuntu-latest",
		Commit:        "abc123",
		FixtureDigest: "fixture-digest",
	}
	if err := UpdateInstallBaseline(dir, result); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "benchmarks", "install-baseline.json"))
	if err != nil {
		t.Fatal(err)
	}
	var baseline InstallBaseline
	if err := json.Unmarshal(raw, &baseline); err != nil {
		t.Fatal(err)
	}
	if baseline.SchemaVersion != installBaselineSchemaVersion {
		t.Fatalf("schemaVersion=%d", baseline.SchemaVersion)
	}
	if len(baseline.Cases) != 1 {
		t.Fatalf("cases=%d", len(baseline.Cases))
	}
	got := baseline.Cases[0]
	if got.TotalMsMedian != 105 || got.TotalMsP95 != 110 || got.FixtureDigest != "fixture-digest" {
		t.Fatalf("%+v", got)
	}

	result.Case = "medium-graph-cold"
	result.MedianMs = 500
	result.P95Ms = 550
	if err := UpdateInstallBaseline(dir, result); err != nil {
		t.Fatal(err)
	}
	raw, err = os.ReadFile(filepath.Join(dir, "benchmarks", "install-baseline.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &baseline); err != nil {
		t.Fatal(err)
	}
	if len(baseline.Cases) != 2 {
		t.Fatalf("cases=%d", len(baseline.Cases))
	}
}
