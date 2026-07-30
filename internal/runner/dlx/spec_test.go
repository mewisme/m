package dlx

import "testing"

func TestRequestIdentityExcludesCommand(t *testing.T) {
	a := RequestIdentity{NormalizedSpecs: []string{"vite"}}
	b := RequestIdentity{NormalizedSpecs: []string{"vite"}, SanitizedRegistryOrigin: "https://registry.npmjs.org"}
	if a.Digest() == b.Digest() {
		t.Fatal("expected different digests for different origins")
	}
}

func TestResolvedEnvironmentExcludesCommand(t *testing.T) {
	env := ResolvedEnvironmentIdentity{GraphDigest: "abc"}
	keyA := NewConsentKey(env, "vite", "vite")
	keyB := NewConsentKey(env, "build", "vite")
	if keyA.Digest() == keyB.Digest() {
		t.Fatal("consent key must include command")
	}
}

func TestParsePackageSpecUnsupported(t *testing.T) {
	if _, err := ParsePackageSpec("file:../x"); err == nil {
		t.Fatal("expected unsupported")
	}
}

func TestInferModeABin(t *testing.T) {
	got, err := InferModeABin("prettier", []string{"prettier"})
	if err != nil || got != "prettier" {
		t.Fatalf("got=%q err=%v", got, err)
	}
	_, err = InferModeABin("typescript", []string{"tsc", "tsserver"})
	if err == nil {
		t.Fatal("expected ambiguous bins error")
	}
}
