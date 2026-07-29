package integration_test

import (
	"strings"
	"testing"
)

const mediumGraphPackageJSON = `{
  "name": "medium-graph",
  "version": "1.0.0",
  "private": true,
  "dependencies": {
    "pkg-a": "1.0.0",
    "lodash": "4.17.21",
    "pkg-cli": "1.0.0",
    "@scope/pkg": "1.0.0"
  }
}`

func TestSBOMMediumGraphSmoke(t *testing.T) {
	projDir, cfgPath, _ := setupRegistryProject(t, mediumGraphPackageJSON)
	if code, out := runM(t, projDir, cfgPath, "install"); code != 0 {
		t.Fatalf("install exit=%d out=%s", code, out)
	}

	code, out := runM(t, projDir, cfgPath, "sbom")
	if code != 0 {
		t.Fatalf("sbom exit=%d out=%s", code, out)
	}
	if !strings.Contains(out, `"bomFormat": "CycloneDX"`) {
		t.Fatalf("expected cyclonedx json, got: %s", out)
	}
	for _, pkg := range []string{"lodash", "pkg-a", "pkg-b", "pkg-c"} {
		if !strings.Contains(out, pkg) {
			t.Fatalf("missing %s in sbom output", pkg)
		}
	}

	code, out = runM(t, projDir, cfgPath, "sbom", "--format", "spdx")
	if code != 0 {
		t.Fatalf("spdx exit=%d out=%s", code, out)
	}
	if !strings.HasPrefix(out, "SPDXVersion: SPDX-2.3") {
		t.Fatalf("expected spdx header, got: %s", out)
	}

	code, out = runM(t, projDir, cfgPath, "sbom", "--redact-internal")
	if code != 0 {
		t.Fatalf("redact exit=%d out=%s", code, out)
	}
	if strings.Contains(out, "@scope/pkg") || strings.Contains(out, "%40scope") {
		t.Fatalf("scope package should be redacted: %s", out)
	}
	if !strings.Contains(out, "[redacted]") {
		t.Fatal("expected redacted placeholder in output")
	}
}
