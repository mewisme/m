package config_test

import (
	"testing"

	"github.com/mewisme/m/internal/config"
)

func TestResolveLinkerModeAutoHoisted(t *testing.T) {
	eff := &config.Effective{Values: map[string]config.Value{}}
	mode, err := config.ResolveLinkerMode(eff, "", false)
	if err != nil || mode != "hoisted" {
		t.Fatalf("got %q err=%v", mode, err)
	}
}

func TestResolveLinkerModeIsolatedRequiresGate(t *testing.T) {
	eff := &config.Effective{Values: map[string]config.Value{
		"install.linker": {Raw: "isolated"},
	}}
	_, err := config.ResolveLinkerMode(eff, "", false)
	if err == nil {
		t.Fatal("expected error without experimental gate")
	}
}

func TestResolveLinkerModeFrozenLockWins(t *testing.T) {
	t.Setenv("MEW_EXPERIMENTAL_ISOLATED_LINKER", "1")
	eff := &config.Effective{Values: map[string]config.Value{
		"install.linker": {Raw: "hoisted"},
	}}
	mode, err := config.ResolveLinkerMode(eff, "isolated", true)
	if err != nil || mode != "isolated" {
		t.Fatalf("got %q err=%v", mode, err)
	}
}
