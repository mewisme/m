package config_test

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/mewisme/m/internal/config"
)

func TestGlobalConfigPathFromEnvMewConfigDir(t *testing.T) {
	env := []string{"MEW_CONFIG_DIR=/custom/config"}
	got := config.GlobalConfigPathFromEnv(env, "linux")
	want, _ := filepath.Abs(filepath.Join("/custom/config", "config.jsonc"))
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestGlobalConfigPathFromEnvMewHome(t *testing.T) {
	env := []string{"MEW_HOME=/mew-home"}
	got := config.GlobalConfigPathFromEnv(env, "linux")
	want, _ := filepath.Abs(filepath.Join("/mew-home", "config", "config.jsonc"))
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestGlobalConfigPathFromEnvXDG(t *testing.T) {
	env := []string{
		"HOME=/home/user",
		"XDG_CONFIG_HOME=/xdg",
	}
	got := config.GlobalConfigPathFromEnv(env, "linux")
	want, _ := filepath.Abs(filepath.Join("/xdg", "mew", "config.jsonc"))
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestGlobalConfigPathFromEnvXDGDefault(t *testing.T) {
	env := []string{"HOME=/home/user"}
	got := config.GlobalConfigPathFromEnv(env, "linux")
	want, _ := filepath.Abs(filepath.Join("/home/user", ".config", "mew", "config.jsonc"))
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestGlobalConfigPathFromEnvWindowsAppData(t *testing.T) {
	env := []string{"AppData=C:\\Users\\me\\AppData\\Roaming"}
	got := config.GlobalConfigPathFromEnv(env, "windows")
	if !strings.HasSuffix(got, filepath.Join("mew", "config.jsonc")) {
		t.Fatalf("unexpected path: %q", got)
	}
	if !strings.Contains(got, "AppData") {
		t.Fatalf("expected AppData in %q", got)
	}
}

func TestGlobalConfigPathFromEnvWindowsUserProfileFallback(t *testing.T) {
	env := []string{"USERPROFILE=C:\\Users\\me"}
	got := config.GlobalConfigPathFromEnv(env, "windows")
	if !strings.Contains(got, "AppData") {
		t.Fatalf("expected AppData fallback in %q", got)
	}
}

func TestGlobalConfigPathFromEnvPrecedence(t *testing.T) {
	env := []string{
		"MEW_CONFIG_DIR=/first",
		"MEW_HOME=/second",
		"HOME=/third",
	}
	got := config.GlobalConfigPathFromEnv(env, "linux")
	want, _ := filepath.Abs(filepath.Join("/first", "config.jsonc"))
	if got != want {
		t.Fatalf("MEW_CONFIG_DIR should win: got %q want %q", got, want)
	}
}

func TestGlobalConfigPathFromEnvNoAmbientGetenv(t *testing.T) {
	// Uses only the snapshot; no os.Getenv in resolver.
	if runtime.GOOS == "windows" {
		t.Setenv("MEW_CONFIG_DIR", "C:\\ambient-should-not-appear")
	} else {
		t.Setenv("MEW_CONFIG_DIR", "/ambient-should-not-appear")
	}
	got := config.GlobalConfigPathFromEnv([]string{"MEW_HOME=/snapshot-only"}, "linux")
	if strings.Contains(got, "ambient") {
		t.Fatalf("resolver used ambient env: %q", got)
	}
	want, _ := filepath.Abs(filepath.Join("/snapshot-only", "config", "config.jsonc"))
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
