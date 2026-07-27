package resolver

import (
	"testing"

	"github.com/mewisme/m/internal/config"
	"github.com/mewisme/m/internal/graph"
	"github.com/mewisme/m/internal/manifest"
	"github.com/mewisme/m/internal/policy"
)

func TestCanonicalizePeerInstancesConflict(t *testing.T) {
	ppc := graph.PeerProviderContext{{Name: "peer-lib", Version: "1.0.0", Key: "peer-lib@1.0.0"}}
	id := graph.PackageID{Name: "plugin-a", Version: "1.0.0", PeerProviderContext: ppc}
	id.Normalize()
	key := id.Key()

	s := &resolveState{
		b: graph.NewBuilder().
			Package(id, "sha512-a", "http://example/a.tgz").
			Package(id, "sha512-b", "http://example/b.tgz"),
		seenPkg: map[string]struct{}{key: {}},
	}
	if err := s.canonicalizePeerInstances(); err == nil {
		t.Fatal("expected identity collision error")
	}
}

func TestPrepareHintsPolicyDriftFromEffectiveConfig(t *testing.T) {
	eff, err := config.Load(t.Context(), config.LoadOptions{
		CWD: t.TempDir(),
		CLI: map[string]any{"resolve.strictPeerDependencies": false},
	})
	if err != nil {
		t.Fatal(err)
	}
	prior, err := graph.NewBuilder().
		Importer(graph.RootImporter, "root").
		Package(graph.PackageID{Name: "lodash", Version: "4.17.21"}, "sha256-l", "").
		EdgeEx(string(graph.RootImporter), "lodash", "lodash@4.17.21", graph.DepProd, "^4.17.0", false).
		Build()
	if err != nil {
		t.Fatal(err)
	}
	m := &manifest.Manifest{
		Name: "root",
		Dependencies: []manifest.Dependency{
			{Name: "lodash", Range: "^4.17.0", Kind: manifest.DepProd},
		},
	}
	strictFP := PolicyFingerprint(&policy.Policy{StrictPeerDependencies: true})
	h := prepareHints(eff, ResolveOptions{
		Prior:             prior,
		IncrementalUpdate: true,
		PriorFingerprints: &PriorFingerprints{ResolverPolicyFingerprint: strictFP},
	}, m)
	if !h.policyDrift {
		t.Fatal("expected policy drift when effective config differs from lock fingerprint")
	}
	if h.canReuse() {
		t.Fatal("policy drift should disable pin reuse")
	}
}

func TestPrepareHintsNoDriftWhenFingerprintsMatch(t *testing.T) {
	eff, err := config.Load(t.Context(), config.LoadOptions{
		CWD: t.TempDir(),
		CLI: map[string]any{"resolve.strictPeerDependencies": true},
	})
	if err != nil {
		t.Fatal(err)
	}
	currentFP := PolicyFingerprint(PolicyFromEffective(eff))
	prior, err := graph.NewBuilder().
		Importer(graph.RootImporter, "root").
		Package(graph.PackageID{Name: "lodash", Version: "4.17.21"}, "sha256-l", "").
		EdgeEx(string(graph.RootImporter), "lodash", "lodash@4.17.21", graph.DepProd, "^4.17.0", false).
		Build()
	if err != nil {
		t.Fatal(err)
	}
	m := &manifest.Manifest{
		Name: "root",
		Dependencies: []manifest.Dependency{
			{Name: "lodash", Range: "^4.17.0", Kind: manifest.DepProd},
		},
	}
	h := prepareHints(eff, ResolveOptions{
		Prior:             prior,
		IncrementalUpdate: true,
		PriorFingerprints: &PriorFingerprints{ResolverPolicyFingerprint: currentFP},
	}, m)
	if h.policyDrift {
		t.Fatal("unexpected policy drift for matching fingerprints")
	}
}

func TestPrepareHintsMissingFingerprintDisablesReuse(t *testing.T) {
	eff, err := config.Load(t.Context(), config.LoadOptions{CWD: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	prior, err := graph.NewBuilder().
		Importer(graph.RootImporter, "root").
		Package(graph.PackageID{Name: "lodash", Version: "4.17.21"}, "sha256-l", "").
		EdgeEx(string(graph.RootImporter), "lodash", "lodash@4.17.21", graph.DepProd, "^4.17.0", false).
		Build()
	if err != nil {
		t.Fatal(err)
	}
	m := &manifest.Manifest{
		Name: "root",
		Dependencies: []manifest.Dependency{
			{Name: "lodash", Range: "^4.17.0", Kind: manifest.DepProd},
		},
	}
	for _, pf := range []*PriorFingerprints{nil, {ResolverPolicyFingerprint: ""}, {ResolverPolicyFingerprint: "not-hex"}} {
		h := prepareHints(eff, ResolveOptions{
			Prior:             prior,
			IncrementalUpdate: true,
			PriorFingerprints: pf,
		}, m)
		if !h.policyDrift {
			t.Fatalf("expected policy drift for %#v", pf)
		}
		if h.canReuse() {
			t.Fatal("missing/malformed fingerprint should disable reuse")
		}
	}
}

func TestPrepareHintsFieldDriftRejectDeprecated(t *testing.T) {
	eff, err := config.Load(t.Context(), config.LoadOptions{
		CWD: t.TempDir(),
		CLI: map[string]any{"resolve.rejectDeprecated": true},
	})
	if err != nil {
		t.Fatal(err)
	}
	currentFP := PolicyFingerprint(PolicyFromEffective(eff))
	looseFP := PolicyFingerprint(&policy.Policy{StrictPeerDependencies: true})
	prior, err := graph.NewBuilder().
		Importer(graph.RootImporter, "root").
		Package(graph.PackageID{Name: "lodash", Version: "4.17.21"}, "sha256-l", "").
		EdgeEx(string(graph.RootImporter), "lodash", "lodash@4.17.21", graph.DepProd, "^4.17.0", false).
		Build()
	if err != nil {
		t.Fatal(err)
	}
	m := &manifest.Manifest{Name: "root", Dependencies: []manifest.Dependency{
		{Name: "lodash", Range: "^4.17.0", Kind: manifest.DepProd},
	}}
	h := prepareHints(eff, ResolveOptions{
		Prior:             prior,
		IncrementalUpdate: true,
		PriorFingerprints: &PriorFingerprints{ResolverPolicyFingerprint: looseFP},
	}, m)
	if !h.policyDrift {
		t.Fatal("expected drift when rejectDeprecated changed")
	}
	h = prepareHints(eff, ResolveOptions{
		Prior:             prior,
		IncrementalUpdate: true,
		PriorFingerprints: &PriorFingerprints{ResolverPolicyFingerprint: currentFP},
	}, m)
	if h.policyDrift {
		t.Fatal("unexpected drift for matching rejectDeprecated fingerprint")
	}
}

func TestPrepareHintsFieldDriftMinimumReleaseAge(t *testing.T) {
	eff, err := config.Load(t.Context(), config.LoadOptions{
		CWD: t.TempDir(),
		CLI: map[string]any{"resolve.minimumReleaseAge": 3600000},
	})
	if err != nil {
		t.Fatal(err)
	}
	currentFP := PolicyFingerprint(PolicyFromEffective(eff))
	zeroFP := PolicyFingerprint(&policy.Policy{StrictPeerDependencies: true})
	prior, err := graph.NewBuilder().
		Importer(graph.RootImporter, "root").
		Package(graph.PackageID{Name: "lodash", Version: "4.17.21"}, "sha256-l", "").
		EdgeEx(string(graph.RootImporter), "lodash", "lodash@4.17.21", graph.DepProd, "^4.17.0", false).
		Build()
	if err != nil {
		t.Fatal(err)
	}
	m := &manifest.Manifest{Name: "root", Dependencies: []manifest.Dependency{
		{Name: "lodash", Range: "^4.17.0", Kind: manifest.DepProd},
	}}
	h := prepareHints(eff, ResolveOptions{
		Prior:             prior,
		IncrementalUpdate: true,
		PriorFingerprints: &PriorFingerprints{ResolverPolicyFingerprint: zeroFP},
	}, m)
	if !h.policyDrift {
		t.Fatal("expected drift when minimumReleaseAge changed")
	}
	h = prepareHints(eff, ResolveOptions{
		Prior:             prior,
		IncrementalUpdate: true,
		PriorFingerprints: &PriorFingerprints{ResolverPolicyFingerprint: currentFP},
	}, m)
	if h.policyDrift {
		t.Fatal("unexpected drift for matching minimumReleaseAge fingerprint")
	}
}

func TestSettingsWithFingerprintsRoundTrip(t *testing.T) {
	eff, err := config.Load(t.Context(), config.LoadOptions{
		CWD: t.TempDir(),
		CLI: map[string]any{
			"resolve.autoInstallPeers":       true,
			"resolve.strictPeerDependencies": false,
			"resolve.rejectDeprecated":       true,
			"resolve.minimumReleaseAge":      60000,
			"offline":                        true,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	pol := PolicyFromEffective(eff)
	want := PolicyFingerprint(pol)
	got := PolicyFingerprint(pol)
	if want != got || want == "" {
		t.Fatalf("fingerprint round trip: %q", got)
	}
}

func TestPrepareHintsIgnoresResolveOptionsPolicyForDrift(t *testing.T) {
	eff, err := config.Load(t.Context(), config.LoadOptions{
		CWD: t.TempDir(),
		CLI: map[string]any{"resolve.strictPeerDependencies": true},
	})
	if err != nil {
		t.Fatal(err)
	}
	prior, err := graph.NewBuilder().
		Importer(graph.RootImporter, "root").
		Package(graph.PackageID{Name: "lodash", Version: "4.17.21"}, "sha256-l", "").
		EdgeEx(string(graph.RootImporter), "lodash", "lodash@4.17.21", graph.DepProd, "^4.17.0", false).
		Build()
	if err != nil {
		t.Fatal(err)
	}
	m := &manifest.Manifest{
		Name: "root",
		Dependencies: []manifest.Dependency{
			{Name: "lodash", Range: "^4.17.0", Kind: manifest.DepProd},
		},
	}
	loose := &policy.Policy{StrictPeerDependencies: false}
	strictFP := PolicyFingerprint(PolicyFromEffective(eff))
	h := prepareHints(eff, ResolveOptions{
		Prior:             prior,
		IncrementalUpdate: true,
		Policy:            loose,
		PriorFingerprints: &PriorFingerprints{ResolverPolicyFingerprint: strictFP},
	}, m)
	if h.policyDrift {
		t.Fatal("drift should compare effective config to lock fingerprint, not opts.Policy")
	}
}
