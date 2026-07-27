package lifecycle_test

import (
	"path/filepath"
	"testing"

	"github.com/mewisme/m/internal/config"
	"github.com/mewisme/m/internal/lifecycle"
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
	if err := lifecycle.CheckTrust("left-pad", eff, store, false, nil, nil); err == nil {
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
	if err := lifecycle.CheckTrust("left-pad", eff, store, false, nil, nil); err != nil {
		t.Fatal(err)
	}
}
