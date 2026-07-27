package mlock_test

import (
	"testing"

	"github.com/mewisme/m/internal/config"
	"github.com/mewisme/m/internal/lockfile/mlock"
)

func TestSettingsFromEffectiveResolvePolicy(t *testing.T) {
	eff := &config.Effective{Values: map[string]config.Value{
		"resolve.autoInstallPeers":       {Raw: true, Source: config.SourceCLI},
		"resolve.strictPeerDependencies": {Raw: false, Source: config.SourceCLI},
	}}
	s, err := mlock.SettingsFromEffective(eff)
	if err != nil {
		t.Fatal(err)
	}
	if !s.Policy.AutoInstallPeers {
		t.Fatal("expected autoInstallPeers true")
	}
	if s.Policy.StrictPeerDependencies {
		t.Fatal("expected strictPeerDependencies false")
	}
}

func TestDefaultSettingsStrictPeers(t *testing.T) {
	s := mlock.DefaultSettings()
	if !s.Policy.StrictPeerDependencies {
		t.Fatal("expected default strictPeerDependencies true")
	}
}
