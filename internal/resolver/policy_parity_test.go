package resolver_test

import (
	"context"
	"testing"
	"time"

	"github.com/mewisme/m/internal/config"
	"github.com/mewisme/m/internal/graph"
	"github.com/mewisme/m/internal/policy"
	"github.com/mewisme/m/internal/resolver"
)

func TestPolicyFromEffectiveLoadsGraphFields(t *testing.T) {
	eff := &config.Effective{Values: map[string]config.Value{
		"resolve.autoInstallPeers":       {Raw: true},
		"resolve.strictPeerDependencies": {Raw: false},
		"resolve.rejectDeprecated":       {Raw: true},
		"resolve.minimumReleaseAge":      {Raw: 3600000},
		"offline":                        {Raw: true},
	}}
	pol := resolver.PolicyFromEffective(eff)
	if !pol.AutoInstallPeers {
		t.Fatal("autoInstallPeers")
	}
	if pol.StrictPeerDependencies {
		t.Fatal("strictPeerDependencies")
	}
	if !pol.RejectDeprecated {
		t.Fatal("rejectDeprecated")
	}
	if pol.MinimumReleaseAge != time.Hour {
		t.Fatalf("minimumReleaseAge=%v want 1h", pol.MinimumReleaseAge)
	}
	if !pol.Offline {
		t.Fatal("offline")
	}
}

func TestResolveProjectUsesEffectivePolicyWhenNil(t *testing.T) {
	eng, _ := testEngine(t)
	root := writeProject(t, `{
  "name": "root",
  "version": "1.0.0",
  "dependencies": { "pkg-c": "1.1.0-beta.1" }
}`)
	eff, err := config.Load(context.Background(), config.LoadOptions{
		CWD: root,
		CLI: map[string]any{"resolve.rejectDeprecated": true},
	})
	if err != nil {
		t.Fatal(err)
	}
	eng.Effective = eff
	_, err = eng.Resolve(context.Background(), root, resolver.ResolveOptions{})
	if err == nil {
		t.Fatal("expected rejectDeprecated via effective config")
	}
}

func TestInstallUpdatePolicyParitySameGraph(t *testing.T) {
	cases := []struct {
		name string
		cli  map[string]any
	}{
		{"strictPeers", map[string]any{"resolve.strictPeerDependencies": true}},
		{"autoInstallPeers", map[string]any{"resolve.autoInstallPeers": true}},
		{"rejectDeprecated", map[string]any{"resolve.rejectDeprecated": false}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			eng, _ := testEngine(t)
			root := writeProject(t, `{
  "name": "root",
  "version": "1.0.0",
  "dependencies": { "lodash": "^4.17.0" }
}`)
			eff, err := config.Load(context.Background(), config.LoadOptions{CWD: root, CLI: tc.cli})
			if err != nil {
				t.Fatal(err)
			}
			eng.Effective = eff
			pol := resolver.PolicyFromEffective(eff)
			fresh, err := eng.Resolve(context.Background(), root, resolver.ResolveOptions{Policy: pol})
			if err != nil {
				t.Fatal(err)
			}
			prior := fresh.Graph
			fp := resolver.PolicyFingerprint(pol)
			incremental, err := eng.Resolve(context.Background(), root, resolver.ResolveOptions{
				Prior:             prior,
				Hints:             prior,
				IncrementalUpdate: true,
				Policy:            pol,
				PriorFingerprints: &resolver.PriorFingerprints{
					ResolverPolicyFingerprint: fp,
					TargetPlatformFingerprint: resolver.TargetPlatformFingerprint(resolver.CurrentTarget()),
				},
			})
			if err != nil {
				t.Fatal(err)
			}
			b1, err := graph.EncodeJSON(fresh.Graph)
			if err != nil {
				t.Fatal(err)
			}
			b2, err := graph.EncodeJSON(incremental.Graph)
			if err != nil {
				t.Fatal(err)
			}
			if string(b1) != string(b2) {
				t.Fatalf("install/update graph mismatch\n--- install ---\n%s\n--- update ---\n%s", b1, b2)
			}
		})
	}
}

func TestPolicyFingerprintIgnoresScriptTrust(t *testing.T) {
	a := resolver.PolicyFingerprint(&policy.Policy{StrictPeerDependencies: true, ScriptTrust: policy.ScriptTrustAllow})
	b := resolver.PolicyFingerprint(&policy.Policy{StrictPeerDependencies: true, ScriptTrust: policy.ScriptTrustDeny})
	if a != b {
		t.Fatalf("script trust should not affect resolver fingerprint: %q vs %q", a, b)
	}
}
