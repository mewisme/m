package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildEnvOverlayNoFlags(t *testing.T) {
	dir := t.TempDir()
	writeEnvTestFile(t, filepath.Join(dir, ".env"), "BASE=from_dotenv\n")

	overlay := buildEnvOverlay(dir, leadingDispatchFlags{})
	if len(overlay) == 0 {
		t.Fatal("expected at least one env var from auto-discovery")
	}
	m := overlayToMap(overlay)
	if m["BASE"] != "from_dotenv" {
		t.Errorf("BASE = %q, want from_dotenv", m["BASE"])
	}
}

func TestBuildEnvOverlayNoEnvFile(t *testing.T) {
	dir := t.TempDir()
	writeEnvTestFile(t, filepath.Join(dir, ".env"), "BASE=should_not_load\n")

	overlay := buildEnvOverlay(dir, leadingDispatchFlags{noEnvFile: true})
	m := overlayToMap(overlay)
	if _, ok := m["BASE"]; ok {
		t.Error("BASE should not be loaded with --no-env-file")
	}
}

func TestBuildEnvOverlayExplicitFile(t *testing.T) {
	dir := t.TempDir()
	writeEnvTestFile(t, filepath.Join(dir, ".env"), "BASE=auto\n")
	explicit := filepath.Join(dir, "custom.env")
	writeEnvTestFile(t, explicit, "BASE=explicit\nCUSTOM=yes\n")

	overlay := buildEnvOverlay(dir, leadingDispatchFlags{envFile: []string{explicit}})
	m := overlayToMap(overlay)
	if m["BASE"] != "explicit" {
		t.Errorf("BASE = %q, want explicit", m["BASE"])
	}
	if m["CUSTOM"] != "yes" {
		t.Errorf("CUSTOM = %q, want yes", m["CUSTOM"])
	}
}

func TestBuildEnvOverlayMode(t *testing.T) {
	dir := t.TempDir()

	overlay := buildEnvOverlay(dir, leadingDispatchFlags{mode: "production"})
	m := overlayToMap(overlay)
	if m["NODE_ENV"] != "production" {
		t.Errorf("NODE_ENV = %q, want production", m["NODE_ENV"])
	}
}

func TestBuildEnvOverlayModeWithDiscovery(t *testing.T) {
	dir := t.TempDir()
	writeEnvTestFile(t, filepath.Join(dir, ".env"), "BASE=from_dotenv\n")
	writeEnvTestFile(t, filepath.Join(dir, ".env.production"), "BASE=from_production\nMODE_SPECIFIC=yes\n")

	overlay := buildEnvOverlay(dir, leadingDispatchFlags{mode: "production"})
	m := overlayToMap(overlay)
	if m["BASE"] != "from_production" {
		t.Errorf("BASE = %q, want from_production (mode-specific should override base)", m["BASE"])
	}
	if m["MODE_SPECIFIC"] != "yes" {
		t.Errorf("MODE_SPECIFIC = %q, want yes", m["MODE_SPECIFIC"])
	}
	if m["NODE_ENV"] != "production" {
		t.Errorf("NODE_ENV = %q, want production", m["NODE_ENV"])
	}
}

func TestBuildEnvOverlayNoEnvFileWithMode(t *testing.T) {
	dir := t.TempDir()
	writeEnvTestFile(t, filepath.Join(dir, ".env"), "BASE=should_not_load\n")

	overlay := buildEnvOverlay(dir, leadingDispatchFlags{noEnvFile: true, mode: "development"})
	m := overlayToMap(overlay)
	if _, ok := m["BASE"]; ok {
		t.Error("BASE should not load with --no-env-file")
	}
	if m["NODE_ENV"] != "development" {
		t.Errorf("NODE_ENV = %q, want development", m["NODE_ENV"])
	}
}

func TestBuildEnvOverlayEmptyDir(t *testing.T) {
	dir := t.TempDir()
	overlay := buildEnvOverlay(dir, leadingDispatchFlags{})
	if len(overlay) != 0 {
		t.Errorf("expected empty overlay for empty dir, got %d entries", len(overlay))
	}
}

func writeEnvTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func overlayToMap(overlay []string) map[string]string {
	m := make(map[string]string, len(overlay))
	for _, kv := range overlay {
		for i := 0; i < len(kv); i++ {
			if kv[i] == '=' {
				m[kv[:i]] = kv[i+1:]
				break
			}
		}
	}
	return m
}
