package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/diagnostics"
	"github.com/mewisme/mew/internal/testkit"
)

func TestExportSBOMMediumGraphSmoke(t *testing.T) {
	testkit.CleanEnv(t)
	moduleRoot := testkit.ModuleRoot(t)
	projectDir := filepath.Join(moduleRoot, ".cache", "mew", "bench", "medium-graph", "project")
	if _, err := os.Stat(filepath.Join(projectDir, "m.lock")); err != nil {
		t.Skip("medium-graph bench project not available")
	}

	ctx := context.Background()
	ac, err := New(ctx, Options{
		CWD:        projectDir,
		Reporter:   diagnostics.NewReporter(diagnostics.Options{Format: "silent"}),
		Version:    "test",
		ConfigPath: filepath.Join(projectDir, "m.jsonc"),
	})
	if err != nil {
		t.Fatal(err)
	}

	cdx, err := ExportSBOM(ctx, ac, ExportSBOMOptions{Format: SBOMCycloneDX})
	if err != nil {
		t.Fatal(err)
	}
	text := string(cdx)
	if !strings.Contains(text, `"bomFormat": "CycloneDX"`) {
		t.Fatalf("unexpected cyclonedx: %s", text)
	}
	for _, pkg := range []string{"lodash", "pkg-a", "pkg-b", "pkg-c"} {
		if !strings.Contains(text, pkg) {
			t.Fatalf("missing %s", pkg)
		}
	}

	spdx, err := ExportSBOM(ctx, ac, ExportSBOMOptions{Format: SBOMSPDX})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(spdx), "SPDXVersion: SPDX-2.3") {
		t.Fatalf("unexpected spdx: %s", spdx)
	}

	redacted, err := ExportSBOM(ctx, ac, ExportSBOMOptions{RedactInternal: true})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(redacted), "@scope/pkg") {
		t.Fatal("scope package should be redacted")
	}
}
