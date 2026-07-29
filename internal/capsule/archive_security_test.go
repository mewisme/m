package capsule_test

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/capsule"
)

const testBlobHex = "9946fe66ac2ea0bcf693bafde3caa98e5760726dfc5298f2a8530a4d528a67f1"

func testManifest(blobHex string) *capsule.Manifest {
	return &capsule.Manifest{
		SchemaVersion: capsule.SchemaVersion,
		ID:            capsule.ComputeID([]byte("lock"), "linux/amd64"),
		CreatedAt:     time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		GraphDigest:   "sha256:graph",
		Platform:      "linux/amd64",
		Lock:          []byte("lock"),
		Manifest:      []byte(`{"name":"demo"}`),
		Blobs: []capsule.BlobRef{{
			Algo: "sha256",
			Hex:  blobHex,
		}},
	}
}

func testBlobBytes() []byte {
	return []byte("tarball-bytes")
}

func createValidArchive(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "valid.capsule")
	err := capsule.Create(context.Background(), capsule.CreateOptions{
		OutputPath: path,
		Manifest:   testManifest(testBlobHex),
		OpenBlob: func(ctx context.Context, ref capsule.BlobRef) (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(testBlobBytes())), nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCreateAtomicNoPartialOutput(t *testing.T) {
	dir := t.TempDir()
	output := filepath.Join(dir, "out.capsule")
	if err := os.WriteFile(output, []byte("prior\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	prior, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	man := testManifest(testBlobHex)
	man.Blobs = append(man.Blobs, capsule.BlobRef{
		Algo: "sha256",
		Hex:  "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	})
	err = capsule.Create(context.Background(), capsule.CreateOptions{
		OutputPath: output,
		Manifest:   man,
		OpenBlob: func(ctx context.Context, ref capsule.BlobRef) (io.ReadCloser, error) {
			if ref.Hex == testBlobHex {
				return io.NopCloser(bytes.NewReader(testBlobBytes())), nil
			}
			return nil, errors.New("blob open failed")
		},
	})
	if err == nil {
		t.Fatal("expected create failure")
	}
	after, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(prior, after) {
		t.Fatal("output file changed after failed create")
	}
	matches, _ := filepath.Glob(filepath.Join(dir, ".*.tmp"))
	if len(matches) > 0 {
		t.Fatalf("left temp files: %v", matches)
	}
}

func TestRestoreRejectsTrailingGarbage(t *testing.T) {
	dir := t.TempDir()
	base := createValidArchive(t, dir)
	data, err := os.ReadFile(base)
	if err != nil {
		t.Fatal(err)
	}
	garbagePath := filepath.Join(dir, "garbage.capsule")
	if err := os.WriteFile(garbagePath, append(data, []byte("TRAILING")...), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = capsule.Restore(context.Background(), capsule.RestoreOptions{
		ArchivePath: garbagePath,
		WriteBlob:   noopWriteBlob,
	})
	if err == nil {
		t.Fatal("expected trailing garbage rejection")
	}
}

func TestRestoreRejectsBadBlobDigest(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad-digest.capsule")
	man := testManifest(testBlobHex)
	if err := capsule.SealIntegrity(man); err != nil {
		t.Fatal(err)
	}
	manifestBytes, err := capsule.EncodeManifestJSON(man)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	if err := writeTarEntry(tw, "capsule.json", manifestBytes); err != nil {
		t.Fatal(err)
	}
	if err := writeTarEntry(tw, man.Blobs[0].BlobPath(), []byte("wrong-bytes")); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	called := false
	_, err = capsule.Restore(context.Background(), capsule.RestoreOptions{
		ArchivePath: path,
		WriteBlob: func(ctx context.Context, ref capsule.BlobRef, r io.Reader) error {
			called = true
			return nil
		},
	})
	if err == nil {
		t.Fatal("expected digest mismatch")
	}
	if called {
		t.Fatal("WriteBlob called after digest failure")
	}
}

func TestRestoreQuarantineNoPublishOnMissingBlob(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "missing-blob.capsule")
	man := testManifest(testBlobHex)
	if err := capsule.SealIntegrity(man); err != nil {
		t.Fatal(err)
	}
	manifestBytes, err := capsule.EncodeManifestJSON(man)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	if err := writeTarEntry(tw, "capsule.json", manifestBytes); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	called := false
	_, err = capsule.Restore(context.Background(), capsule.RestoreOptions{
		ArchivePath: path,
		WriteBlob: func(ctx context.Context, ref capsule.BlobRef, r io.Reader) error {
			called = true
			return nil
		},
	})
	if err == nil {
		t.Fatal("expected missing blob error")
	}
	if called {
		t.Fatal("WriteBlob called before all blobs validated")
	}
}

func TestRestoreRejectsUnexpectedBlob(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "extra-blob.capsule")
	man := testManifest(testBlobHex)
	if err := capsule.SealIntegrity(man); err != nil {
		t.Fatal(err)
	}
	manifestBytes, err := capsule.EncodeManifestJSON(man)
	if err != nil {
		t.Fatal(err)
	}
	extra := capsule.BlobRef{
		Algo: "sha256",
		Hex:  "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	}
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	if err := writeTarEntry(tw, "capsule.json", manifestBytes); err != nil {
		t.Fatal(err)
	}
	if err := writeTarEntry(tw, man.Blobs[0].BlobPath(), testBlobBytes()); err != nil {
		t.Fatal(err)
	}
	if err := writeTarEntry(tw, extra.BlobPath(), []byte("extra")); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	called := false
	_, err = capsule.Restore(context.Background(), capsule.RestoreOptions{
		ArchivePath: path,
		WriteBlob: func(ctx context.Context, ref capsule.BlobRef, r io.Reader) error {
			called = true
			return nil
		},
	})
	if err == nil {
		t.Fatal("expected unexpected blob error")
	}
	if called {
		t.Fatal("WriteBlob called when unexpected blob present")
	}
}

func TestRestoreRejectsDuplicateBlob(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "dup-blob.capsule")
	man := testManifest(testBlobHex)
	if err := capsule.SealIntegrity(man); err != nil {
		t.Fatal(err)
	}
	manifestBytes, err := capsule.EncodeManifestJSON(man)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	if err := writeTarEntry(tw, "capsule.json", manifestBytes); err != nil {
		t.Fatal(err)
	}
	if err := writeTarEntry(tw, man.Blobs[0].BlobPath(), testBlobBytes()); err != nil {
		t.Fatal(err)
	}
	if err := writeTarEntry(tw, man.Blobs[0].BlobPath(), testBlobBytes()); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = capsule.Restore(context.Background(), capsule.RestoreOptions{
		ArchivePath: path,
		WriteBlob:   noopWriteBlob,
	})
	if err == nil {
		t.Fatal("expected duplicate blob error")
	}
}

func TestRestoreRejectsTamperedManifest(t *testing.T) {
	dir := t.TempDir()
	_ = createValidArchive(t, dir)
	man := testManifest(testBlobHex)
	if err := capsule.SealIntegrity(man); err != nil {
		t.Fatal(err)
	}
	man.Lock = []byte("tampered-lock")
	manifestBytes, err := capsule.EncodeManifestJSON(man)
	if err != nil {
		t.Fatal(err)
	}
	tamperPath := filepath.Join(dir, "tampered.capsule")
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	if err := writeTarEntry(tw, "capsule.json", manifestBytes); err != nil {
		t.Fatal(err)
	}
	if err := writeTarEntry(tw, man.Blobs[0].BlobPath(), testBlobBytes()); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tamperPath, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = capsule.Restore(context.Background(), capsule.RestoreOptions{
		ArchivePath: tamperPath,
		WriteBlob:   noopWriteBlob,
	})
	if err == nil {
		t.Fatal("expected integrity failure")
	}
	var ae *apperr.Error
	if !errors.As(err, &ae) || ae.Code != apperr.Integrity {
		t.Fatalf("expected integrity error, got %v", err)
	}
}

func TestRestorePublishesOnlyAfterValidation(t *testing.T) {
	dir := t.TempDir()
	path := createValidArchive(t, dir)
	var published []string
	restored, err := capsule.Restore(context.Background(), capsule.RestoreOptions{
		ArchivePath: path,
		WriteBlob: func(ctx context.Context, ref capsule.BlobRef, r io.Reader) error {
			data, err := io.ReadAll(r)
			if err != nil {
				return err
			}
			sum := sha256.Sum256(data)
			if hex.EncodeToString(sum[:]) != ref.Hex {
				return errors.New("unexpected blob at publish")
			}
			published = append(published, ref.Algo+"/"+ref.Hex)
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(published) != 1 || published[0] != "sha256/"+testBlobHex {
		t.Fatalf("publish set: %v", published)
	}
	if restored.ID != testManifest(testBlobHex).ID {
		t.Fatal("manifest id mismatch")
	}
}

func noopWriteBlob(ctx context.Context, ref capsule.BlobRef, r io.Reader) error {
	_, err := io.Copy(io.Discard, r)
	return err
}

func writeTarEntry(tw *tar.Writer, name string, data []byte) error {
	hdr := &tar.Header{
		Name:   name,
		Mode:   0o644,
		Size:   int64(len(data)),
		Format: tar.FormatUSTAR,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	_, err := tw.Write(data)
	return err
}
