package app

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mewisme/mew/internal/diagnostics"
	"github.com/mewisme/mew/internal/testkit"
)

func TestBenchInstallMediumGraphCold(t *testing.T) {
	testkit.CleanEnv(t)
	moduleRoot := testkit.ModuleRoot(t)
	ctx := context.Background()
	ac, err := New(ctx, Options{
		CWD:      moduleRoot,
		Reporter: diagnostics.NewReporter(diagnostics.Options{Format: "silent"}),
		Version:  "test",
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := BenchInstall(ctx, ac, BenchInstallOptions{Mode: BenchCold})
	if err != nil {
		t.Fatal(err)
	}
	if result.Case != "medium-graph-cold" {
		t.Fatalf("case=%s", result.Case)
	}
	if result.TotalMs <= 0 {
		t.Fatalf("totalMs=%d", result.TotalMs)
	}
	if len(result.Phases) == 0 {
		t.Fatal("expected phase timings")
	}
	if _, err := os.Stat(filepath.Join(moduleRoot, ".cache", "mew", "bench", "medium-graph", "project", "node_modules")); err != nil {
		t.Fatal(err)
	}

	data, err := EncodeBenchResultJSON(result)
	if err != nil {
		t.Fatal(err)
	}
	var round BenchResult
	if err := json.Unmarshal(data, &round); err != nil {
		t.Fatal(err)
	}
	if round.Mode != "cold" || round.TotalMs != result.TotalMs {
		t.Fatalf("%+v", round)
	}
}

func TestPhaseReporterAccumulates(t *testing.T) {
	rep := newPhaseReporter(diagnostics.NewReporter(diagnostics.Options{Format: "silent"}))
	rep.Progress(diagnostics.Event{Type: "progress", Phase: "resolve"})
	time.Sleep(2 * time.Millisecond)
	rep.Progress(diagnostics.Event{Type: "progress", Phase: "fetch"})
	time.Sleep(2 * time.Millisecond)
	rep.Progress(diagnostics.Event{Type: "progress", Phase: "link"})
	phases := rep.finish()
	if len(phases) < 2 {
		t.Fatalf("phases=%v", phases)
	}
	if rep.totalMs() <= 0 {
		t.Fatal("expected positive totalMs")
	}
}
