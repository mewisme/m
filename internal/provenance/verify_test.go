package provenance

import (
	"context"
	"crypto/ed25519"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fixtureVerifyOpts() VerifyOptions {
	return VerifyOptions{TrustPolicy: TrustFixtureKey}
}

func fixturePrivateKey() ed25519.PrivateKey {
	seed := make([]byte, ed25519.SeedSize)
	copy(seed, []byte("mew-mvp0030-provenance-fixture"))
	return ed25519.NewKeyFromSeed(seed)
}

func TestFixturePublicKeyMatchesPrivate(t *testing.T) {
	priv := fixturePrivateKey()
	pub := FixturePublicKey()
	if !priv.Public().(ed25519.PublicKey).Equal(pub) {
		t.Fatal("fixture key pair mismatch")
	}
}

func TestVerifySignedFixture(t *testing.T) {
	path := filepath.Join("..", "..", "fixtures", "provenance", "signed-pkg", "signed-fixture-pkg-1.0.0.attestation.json")
	metaPath := filepath.Join("..", "..", "fixtures", "provenance", "signed-pkg", "metadata.json")
	metaRaw, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatal(err)
	}
	var meta struct {
		TarballSHA512 string `json:"tarballSha512"`
	}
	if err := json.Unmarshal(metaRaw, &meta); err != nil {
		t.Fatal(err)
	}
	res, err := Verify(context.Background(), path, TarballDigest{Algo: "sha512", Hex: meta.TarballSHA512}, fixtureVerifyOpts())
	if err != nil {
		t.Fatal(err)
	}
	if res.PackageName != "signed-fixture-pkg" || res.PackageVersion != "1.0.0" {
		t.Fatalf("package %+v", res)
	}
	if res.TrustSource != "fixture" || res.VerificationMode != "digest-only" {
		t.Fatalf("trust/mode %+v", res)
	}
}

func TestVerifyRejectsMissingTrustPolicy(t *testing.T) {
	bundle := signTestBundle(t, "signed-fixture-pkg", "1.0.0", "abc123")
	dir := t.TempDir()
	path := filepath.Join(dir, "bundle.json")
	writeBundle(t, path, bundle)
	_, err := Verify(context.Background(), path, TarballDigest{}, VerifyOptions{})
	if err == nil || !strings.Contains(err.Error(), "trust policy required") {
		t.Fatalf("expected trust policy error, got %v", err)
	}
}

func TestVerifyConfiguredTrustRequiresKey(t *testing.T) {
	bundle := signTestBundle(t, "signed-fixture-pkg", "1.0.0", "abc123")
	dir := t.TempDir()
	path := filepath.Join(dir, "bundle.json")
	writeBundle(t, path, bundle)
	_, err := Verify(context.Background(), path, TarballDigest{}, VerifyOptions{TrustPolicy: TrustConfiguredKey})
	if err == nil || !strings.Contains(err.Error(), "configured trust requires trusted public key") {
		t.Fatalf("expected configured key error, got %v", err)
	}
}

func TestVerifySigstoreRootsUnsupported(t *testing.T) {
	bundle := signTestBundle(t, "signed-fixture-pkg", "1.0.0", "abc123")
	dir := t.TempDir()
	path := filepath.Join(dir, "bundle.json")
	writeBundle(t, path, bundle)
	_, err := Verify(context.Background(), path, TarballDigest{}, VerifyOptions{TrustPolicy: TrustSigstoreRoots})
	if err == nil || !strings.Contains(err.Error(), "sigstore root trust is not supported") {
		t.Fatalf("expected sigstore unsupported error, got %v", err)
	}
}

func TestVerifyBindingMatchesCorrectSubjectAmongMany(t *testing.T) {
	priv := fixturePrivateKey()
	bundle := signBundleStatement(Statement{
		Type:          statementTypeV1,
		PredicateType: predicateSLSAV1,
		Subject: []Subject{
			{Name: "pkg:npm/pkg-a@1.0.0", Digest: map[string]string{"sha512": "aaa111"}},
			{Name: "pkg:npm/pkg-b@2.0.0", Digest: map[string]string{"sha512": "bbb222"}},
		},
	}, priv)
	dir := t.TempDir()
	path := filepath.Join(dir, "bundle.json")
	writeBundle(t, path, bundle)
	bind := BindingFromNameVersion("pkg-a", "1.0.0", TarballDigest{Algo: "sha512", Hex: "aaa111"})
	opts := fixtureVerifyOpts()
	opts.Binding = &bind
	res, err := Verify(context.Background(), path, TarballDigest{Algo: "sha512", Hex: "aaa111"}, opts)
	if err != nil {
		t.Fatal(err)
	}
	if res.PackageName != "pkg-a" || res.VerificationMode != "binding" {
		t.Fatalf("result %+v", res)
	}
}

func TestVerifyBindingRejectsNoMatchingSubject(t *testing.T) {
	bundle := signTestBundle(t, "pkg-a", "1.0.0", "aaa111")
	bind := BindingFromNameVersion("pkg-b", "2.0.0", TarballDigest{Algo: "sha512", Hex: "bbb222"})
	opts := fixtureVerifyOpts()
	opts.Binding = &bind
	dir := t.TempDir()
	path := filepath.Join(dir, "bundle.json")
	writeBundle(t, path, bundle)
	_, err := Verify(context.Background(), path, TarballDigest{}, opts)
	if err == nil || !strings.Contains(err.Error(), "no attestation subject matches package") {
		t.Fatalf("expected no match error, got %v", err)
	}
}

func TestVerifyBindingRejectsAmbiguousSubjects(t *testing.T) {
	priv := fixturePrivateKey()
	bundle := signBundleStatement(Statement{
		Type:          statementTypeV1,
		PredicateType: predicateSLSAV1,
		Subject: []Subject{
			{Name: "pkg:npm/pkg-a@1.0.0", Digest: map[string]string{"sha512": "aaa111"}},
			{Name: "pkg:npm/pkg-a@1.0.0", Digest: map[string]string{"sha512": "aaa111"}},
		},
	}, priv)
	dir := t.TempDir()
	path := filepath.Join(dir, "bundle.json")
	writeBundle(t, path, bundle)
	bind := BindingFromNameVersion("pkg-a", "1.0.0", TarballDigest{Algo: "sha512", Hex: "aaa111"})
	opts := fixtureVerifyOpts()
	opts.Binding = &bind
	_, err := Verify(context.Background(), path, TarballDigest{}, opts)
	if err == nil || !strings.Contains(err.Error(), "ambiguous attestation subjects") {
		t.Fatalf("expected ambiguous subject error, got %v", err)
	}
}

func TestVerifyRejectsMultipleSubjectsWithoutBinding(t *testing.T) {
	priv := fixturePrivateKey()
	bundle := signBundleStatement(Statement{
		Type:          statementTypeV1,
		PredicateType: predicateSLSAV1,
		Subject: []Subject{
			{Name: "pkg:npm/pkg-a@1.0.0", Digest: map[string]string{"sha512": "aaa111"}},
			{Name: "pkg:npm/pkg-b@2.0.0", Digest: map[string]string{"sha512": "bbb222"}},
		},
	}, priv)
	dir := t.TempDir()
	path := filepath.Join(dir, "bundle.json")
	writeBundle(t, path, bundle)
	_, err := Verify(context.Background(), path, TarballDigest{}, fixtureVerifyOpts())
	if err == nil || !strings.Contains(err.Error(), "package binding required") {
		t.Fatalf("expected binding required error, got %v", err)
	}
}

func TestVerifyRejectsTamperedSignature(t *testing.T) {
	bundle := signTestBundle(t, "signed-fixture-pkg", "1.0.0", "abc123")
	bundle.DSSEEnvelope.Signatures[0].Sig = base64.StdEncoding.EncodeToString(make([]byte, ed25519.SignatureSize))
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	writeBundle(t, path, bundle)
	_, err := Verify(context.Background(), path, TarballDigest{Algo: "sha512", Hex: "abc123"}, fixtureVerifyOpts())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestVerifyRejectsDigestMismatch(t *testing.T) {
	bundle := signTestBundle(t, "signed-fixture-pkg", "1.0.0", "abc123")
	dir := t.TempDir()
	path := filepath.Join(dir, "bundle.json")
	writeBundle(t, path, bundle)
	_, err := Verify(context.Background(), path, TarballDigest{Algo: "sha512", Hex: "deadbeef"}, fixtureVerifyOpts())
	if err == nil {
		t.Fatal("expected error")
	}
}

func signTestBundle(t *testing.T, name, version, sha512Hex string) Bundle {
	t.Helper()
	return signBundle(name, version, sha512Hex, fixturePrivateKey())
}

func signBundle(name, version, sha512Hex string, priv ed25519.PrivateKey) Bundle {
	return signBundleStatement(Statement{
		Type:          statementTypeV1,
		PredicateType: predicateSLSAV1,
		Subject: []Subject{{
			Name:   "pkg:npm/" + name + "@" + version,
			Digest: map[string]string{"sha512": sha512Hex},
		}},
	}, priv)
}

func signBundleStatement(stmt Statement, priv ed25519.PrivateKey) Bundle {
	stmtRaw, _ := json.Marshal(stmt)
	payload := base64.StdEncoding.EncodeToString(stmtRaw)
	pae := dssePAE(payloadTypeInToto, payload)
	sig := ed25519.Sign(priv, pae)
	pub := priv.Public().(ed25519.PublicKey)
	return Bundle{
		MediaType: mediaTypeBundleV03,
		VerificationMaterial: VerificationMaterial{
			PublicKey: &PublicKey{RawBytes: base64.StdEncoding.EncodeToString(pub)},
		},
		DSSEEnvelope: DSSEEnvelope{
			PayloadType: payloadTypeInToto,
			Payload:     payload,
			Signatures:  []DSSESig{{Sig: base64.StdEncoding.EncodeToString(sig)}},
		},
	}
}

func writeBundle(t *testing.T, path string, bundle Bundle) {
	t.Helper()
	raw, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestWriteSignedPkgFixture(t *testing.T) {
	if os.Getenv("MEW_WRITE_PROVENANCE_FIXTURE") == "" {
		t.Skip("set MEW_WRITE_PROVENANCE_FIXTURE=1 to regenerate fixture attestations")
	}
	tgzPath := filepath.Join("..", "..", "fixtures", "provenance", "signed-pkg", "signed-fixture-pkg-1.0.0.tgz")
	data, err := os.ReadFile(tgzPath)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha512.Sum512(data)
	hexDigest := hex.EncodeToString(sum[:])
	bundle := signBundle("signed-fixture-pkg", "1.0.0", hexDigest, fixturePrivateKey())
	attPath := filepath.Join("..", "..", "fixtures", "provenance", "signed-pkg", "signed-fixture-pkg-1.0.0.attestation.json")
	writeBundle(t, attPath, bundle)
	metaPath := filepath.Join("..", "..", "fixtures", "provenance", "signed-pkg", "metadata.json")
	meta := map[string]string{"tarballSha512": hexDigest}
	raw, _ := json.MarshalIndent(meta, "", "  ")
	raw = append(raw, '\n')
	if err := os.WriteFile(metaPath, raw, 0o644); err != nil {
		t.Fatal(err)
	}
}
