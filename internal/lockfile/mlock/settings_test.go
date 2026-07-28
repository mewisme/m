package mlock_test

import (
	"testing"

	"github.com/mewisme/mew/internal/config"
	"github.com/mewisme/mew/internal/lockfile/mlock"
	"github.com/mewisme/mew/internal/resolver"
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

func TestSettingsWithFingerprintsUsesResolverEncoder(t *testing.T) {
	eff := &config.Effective{Values: map[string]config.Value{
		"resolve.autoInstallPeers":       {Raw: true, Source: config.SourceCLI},
		"resolve.strictPeerDependencies": {Raw: true, Source: config.SourceCLI},
	}}
	s, err := mlock.SettingsWithFingerprints(eff, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := resolver.PolicyFingerprint(resolver.PolicyFromEffective(eff))
	if s.ResolverPolicyFingerprint != want {
		t.Fatalf("fingerprint=%q want %q", s.ResolverPolicyFingerprint, want)
	}
}

func TestDefaultSettingsStrictPeers(t *testing.T) {
	s := mlock.DefaultSettings()
	if !s.Policy.StrictPeerDependencies {
		t.Fatal("expected default strictPeerDependencies true")
	}
}
