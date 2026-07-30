package lifecycle_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/config"
	"github.com/mewisme/mew/internal/lifecycle"
	"github.com/mewisme/mew/internal/prompt"
)

func TestTrustRoundTrip(t *testing.T) {
	root := t.TempDir()
	store, err := lifecycle.LoadTrust(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.AddTrusted("esbuild"); err != nil {
		t.Fatal(err)
	}
	loaded, err := lifecycle.LoadTrust(root)
	if err != nil {
		t.Fatal(err)
	}
	if !loaded.IsTrusted("esbuild") {
		t.Fatal("expected esbuild trusted")
	}
	if _, err := filepath.Abs(root); err != nil {
		t.Fatal(err)
	}
}

func TestCheckTrustDenyBlocksUnknown(t *testing.T) {
	eff := &config.Effective{Values: map[string]config.Value{
		"lifecycle.script_trust": {Raw: "deny"},
	}}
	store, _ := lifecycle.LoadTrust(t.TempDir())
	if err := lifecycle.CheckTrust(context.Background(), "left-pad", eff, store, false, nil, nil); err == nil {
		t.Fatal("expected deny")
	}
}

func TestCheckTrustAllowsTrusted(t *testing.T) {
	eff := &config.Effective{Values: map[string]config.Value{
		"lifecycle.script_trust": {Raw: "deny"},
	}}
	root := t.TempDir()
	store, _ := lifecycle.LoadTrust(root)
	if err := store.AddTrusted("left-pad"); err != nil {
		t.Fatal(err)
	}
	if err := lifecycle.CheckTrust(context.Background(), "left-pad", eff, store, false, nil, nil); err != nil {
		t.Fatal(err)
	}
}

func TestCheckTrustAskNonInteractiveFailsClosed(t *testing.T) {
	eff := &config.Effective{Values: map[string]config.Value{
		"lifecycle.script_trust": {Raw: "ask"},
	}}
	store, _ := lifecycle.LoadTrust(t.TempDir())
	if err := lifecycle.CheckTrust(context.Background(), "esbuild", eff, store, false, nil, nil); err == nil {
		t.Fatal("expected policy error")
	}
}

func TestCheckTrustAskTrustProjectPersists(t *testing.T) {
	root := t.TempDir()
	store, _ := lifecycle.LoadTrust(root)
	eff := &config.Effective{Values: map[string]config.Value{
		"lifecycle.script_trust": {Raw: "ask"},
	}}
	p := &prompt.ScriptedPrompter{Answers: []prompt.PromptAnswer{{OptionID: prompt.OptionTrustProject}}}
	if err := lifecycle.CheckTrust(context.Background(), "esbuild", eff, store, true, p, map[string]struct{}{}); err != nil {
		t.Fatal(err)
	}
	loaded, err := lifecycle.LoadTrust(root)
	if err != nil {
		t.Fatal(err)
	}
	if !loaded.IsTrusted("esbuild") {
		t.Fatal("expected persisted trust")
	}
}

func TestCheckTrustAskAllowOnceNotPersisted(t *testing.T) {
	root := t.TempDir()
	store, _ := lifecycle.LoadTrust(root)
	eff := &config.Effective{Values: map[string]config.Value{
		"lifecycle.script_trust": {Raw: "ask"},
	}}
	once := map[string]struct{}{}
	p := &prompt.ScriptedPrompter{Answers: []prompt.PromptAnswer{{OptionID: prompt.OptionAllowOnce}}}
	if err := lifecycle.CheckTrust(context.Background(), "esbuild", eff, store, true, p, once); err != nil {
		t.Fatal(err)
	}
	if _, ok := once["esbuild"]; !ok {
		t.Fatal("expected allow-once session entry")
	}
	loaded, err := lifecycle.LoadTrust(root)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.IsTrusted("esbuild") {
		t.Fatal("allow-once must not persist")
	}
	// Second check without prompt uses allow-once map.
	if err := lifecycle.CheckTrust(context.Background(), "esbuild", eff, store, false, nil, once); err != nil {
		t.Fatal(err)
	}
}

func TestCheckTrustAskDeny(t *testing.T) {
	eff := &config.Effective{Values: map[string]config.Value{
		"lifecycle.script_trust": {Raw: "ask"},
	}}
	store, _ := lifecycle.LoadTrust(t.TempDir())
	p := &prompt.ScriptedPrompter{Answers: []prompt.PromptAnswer{{OptionID: prompt.OptionDeny}}}
	if err := lifecycle.CheckTrust(context.Background(), "esbuild", eff, store, true, p, nil); err == nil {
		t.Fatal("expected deny")
	}
}
