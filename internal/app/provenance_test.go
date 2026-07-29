package app

import (
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/config"
	"github.com/mewisme/mew/internal/graph"
	"github.com/mewisme/mew/internal/provenance"
)

func TestPackageFromGraphExactKey(t *testing.T) {
	g := &graph.Graph{
		Packages: []graph.Package{
			{ID: graph.PackageID{Name: "pkg-a", Version: "1.0.0"}},
			{ID: graph.PackageID{Name: "pkg-a", Version: "2.0.0"}},
		},
	}
	pkg, err := packageFromGraphKey(g, "pkg-a@2.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if pkg.ID.Version != "2.0.0" {
		t.Fatalf("got %+v", pkg.ID)
	}
}

func TestPackageFromGraphAmbiguousNameOnly(t *testing.T) {
	g := &graph.Graph{
		Packages: []graph.Package{
			{ID: graph.PackageID{Name: "pkg-a", Version: "1.0.0"}},
			{ID: graph.PackageID{Name: "pkg-a", Version: "2.0.0"}},
		},
	}
	_, err := packageFromGraphKey(g, "pkg-a")
	if err == nil {
		t.Fatal("expected ambiguous error")
	}
	if apperr.CodeOf(err) != apperr.LockAmbiguous {
		t.Fatalf("code=%s err=%v", apperr.CodeOf(err), err)
	}
}

func TestProvenanceTrustFromConfigRequiresKey(t *testing.T) {
	ac := &Context{Config: &config.Effective{Values: map[string]config.Value{}}}
	_, err := provenanceTrustFromConfig(ac)
	if err == nil {
		t.Fatal("expected config error")
	}
	if apperr.CodeOf(err) != apperr.Config {
		t.Fatalf("code=%s err=%v", apperr.CodeOf(err), err)
	}
	if !strings.Contains(err.Error(), "trusted public key not configured") {
		t.Fatalf("err=%v", err)
	}
}

func TestProvenanceTrustFromConfigParsesKey(t *testing.T) {
	ac := &Context{Config: &config.Effective{Values: map[string]config.Value{
		"provenance.trusted_public_key": {Raw: provenance.FixturePublicKeyBase64()},
	}}}
	opts, err := provenanceTrustFromConfig(ac)
	if err != nil {
		t.Fatal(err)
	}
	if opts.TrustPolicy != provenance.TrustConfiguredKey {
		t.Fatalf("policy=%v", opts.TrustPolicy)
	}
	if opts.TrustedPublicKey == nil {
		t.Fatal("expected public key")
	}
}
