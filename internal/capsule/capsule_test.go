package capsule_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mewisme/mew/internal/capsule"
)

func TestCapsuleSortsPackages(t *testing.T) {
	c := &capsule.Capsule{
		SchemaVersion: capsule.SchemaVersion,
		ID:            "cap-1",
		Packages:      []string{"z@1.0.0", "a@1.0.0"},
		Integrity:     "sha512-x",
	}
	first, err := capsule.EncodeJSON(c)
	if err != nil {
		t.Fatal(err)
	}
	again, err := capsule.DecodeJSON(first)
	if err != nil {
		t.Fatal(err)
	}
	if again.Packages[0] != "a@1.0.0" {
		t.Fatalf("unsorted: %v", again.Packages)
	}
	second, err := capsule.EncodeJSON(again)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatalf("unstable")
	}
}

func TestManifestIntegrityRoundTrip(t *testing.T) {
	man := &capsule.Manifest{
		SchemaVersion: capsule.SchemaVersion,
		ID:            "sha256:abc",
		CreatedAt:     time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		GraphDigest:   "sha256:graph",
		Platform:      "linux/amd64",
		Lock:          []byte("lock"),
		Manifest:      []byte(`{"name":"demo"}`),
		Blobs: []capsule.BlobRef{
			{Algo: "sha256", Hex: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
			{Algo: "sha256", Hex: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		},
	}
	if err := capsule.SealIntegrity(man); err != nil {
		t.Fatal(err)
	}
	raw, err := capsule.EncodeManifestJSON(man)
	if err != nil {
		t.Fatal(err)
	}
	got, err := capsule.DecodeManifestJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := capsule.VerifyIntegrity(got); err != nil {
		t.Fatal(err)
	}
	if got.Blobs[0].Hex != man.Blobs[0].Hex {
		t.Fatalf("blob sort changed: %v", got.Blobs)
	}
}

func TestArchiveCreateRestore(t *testing.T) {
	blob := []byte("tarball-bytes")
	man := &capsule.Manifest{
		SchemaVersion: capsule.SchemaVersion,
		ID:            capsule.ComputeID([]byte("lock"), "linux/amd64"),
		CreatedAt:     time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		GraphDigest:   "sha256:graph",
		Platform:      "linux/amd64",
		Lock:          []byte("lock"),
		Manifest:      []byte(`{"name":"demo"}`),
		Blobs: []capsule.BlobRef{{
			Algo: "sha256",
			Hex:  "9946fe66ac2ea0bcf693bafde3caa98e5760726dfc5298f2a8530a4d528a67f1",
		}},
	}
	archivePath := filepath.Join(t.TempDir(), "demo.capsule")
	err := capsule.Create(context.Background(), capsule.CreateOptions{
		OutputPath: archivePath,
		Manifest:   man,
		OpenBlob: func(ctx context.Context, ref capsule.BlobRef) (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(blob)), nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	written := map[string][]byte{}
	restored, err := capsule.Restore(context.Background(), capsule.RestoreOptions{
		ArchivePath: archivePath,
		WriteBlob: func(ctx context.Context, ref capsule.BlobRef, r io.Reader) error {
			data, err := io.ReadAll(r)
			if err != nil {
				return err
			}
			written[ref.Algo+"/"+ref.Hex] = data
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(restored.Lock, man.Lock) {
		t.Fatal("lock mismatch")
	}
	if !bytes.Equal(written["sha256/9946fe66ac2ea0bcf693bafde3caa98e5760726dfc5298f2a8530a4d528a67f1"], blob) {
		t.Fatal("blob mismatch")
	}
	if _, err := os.Stat(archivePath); err != nil {
		t.Fatal(err)
	}
}
