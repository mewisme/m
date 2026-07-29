package capsule

import (
	"archive/tar"
	"context"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/contentid"
	"github.com/mewisme/mew/internal/fsx"
)

const (
	manifestArchiveName = "capsule.json"
	maxManifestBytes    = 32 << 20  // 32 MiB
	maxBlobBytes        = 512 << 20 // 512 MiB; matches store verified cap
)

// BlobOpener returns one blob payload for archive creation.
type BlobOpener func(ctx context.Context, ref BlobRef) (io.ReadCloser, error)

// BlobWriter stores one blob payload during restore.
type BlobWriter func(ctx context.Context, ref BlobRef, r io.Reader) error

// CreateOptions configures capsule archive creation.
type CreateOptions struct {
	OutputPath string
	Manifest   *Manifest
	OpenBlob   BlobOpener
}

// RestoreOptions configures capsule archive restore.
type RestoreOptions struct {
	ArchivePath string
	WriteBlob   BlobWriter
}

// Create writes capsule.json and referenced blobs to a tar archive.
func Create(ctx context.Context, opts CreateOptions) error {
	if opts.Manifest == nil {
		return apperr.New(apperr.Internal, "capsule.create", opts.OutputPath, "nil manifest")
	}
	if opts.OutputPath == "" {
		return apperr.New(apperr.Usage, "capsule.create", "", "empty output path")
	}
	if opts.OpenBlob == nil {
		return apperr.New(apperr.Internal, "capsule.create", opts.OutputPath, "nil blob opener")
	}
	if err := SealIntegrity(opts.Manifest); err != nil {
		return err
	}
	manifestBytes, err := EncodeManifestJSON(opts.Manifest)
	if err != nil {
		return err
	}
	outputDir := filepath.Dir(opts.OutputPath)
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return apperr.Wrap(apperr.IO, "capsule.create", opts.OutputPath, err)
	}
	tmp, err := os.CreateTemp(outputDir, "."+filepath.Base(opts.OutputPath)+".*.tmp")
	if err != nil {
		return apperr.Wrap(apperr.IO, "capsule.create", opts.OutputPath, err)
	}
	tmpPath := tmp.Name()
	failed := true
	defer func() {
		_ = tmp.Close()
		if failed {
			_ = os.Remove(tmpPath)
		}
	}()
	tw := tar.NewWriter(tmp)
	if err := writeTarBytes(tw, manifestArchiveName, manifestBytes); err != nil {
		return err
	}
	for _, ref := range opts.Manifest.Blobs {
		if err := ctx.Err(); err != nil {
			return err
		}
		rc, err := opts.OpenBlob(ctx, ref)
		if err != nil {
			return err
		}
		err = writeTarStream(tw, ref.BlobPath(), rc)
		_ = rc.Close()
		if err != nil {
			return err
		}
	}
	if err := tw.Close(); err != nil {
		return apperr.Wrap(apperr.IO, "capsule.create", opts.OutputPath, err)
	}
	if err := tmp.Sync(); err != nil {
		return apperr.Wrap(apperr.IO, "capsule.create", opts.OutputPath, err)
	}
	if err := tmp.Close(); err != nil {
		return apperr.Wrap(apperr.IO, "capsule.create", opts.OutputPath, err)
	}
	if err := fsx.ReplaceFileRecoverable(tmpPath, opts.OutputPath); err != nil {
		return apperr.Wrap(apperr.IO, "capsule.create", opts.OutputPath, err)
	}
	failed = false
	return nil
}

// Restore reads a capsule archive, verifies manifest integrity, and writes blobs.
func Restore(ctx context.Context, opts RestoreOptions) (*Manifest, error) {
	if opts.ArchivePath == "" {
		return nil, apperr.New(apperr.Usage, "capsule.restore", "", "empty archive path")
	}
	if opts.WriteBlob == nil {
		return nil, apperr.New(apperr.Internal, "capsule.restore", opts.ArchivePath, "nil blob writer")
	}
	f, err := os.Open(opts.ArchivePath)
	if err != nil {
		return nil, apperr.Wrap(apperr.IO, "capsule.restore", opts.ArchivePath, err)
	}
	defer func() { _ = f.Close() }()

	quarantineDir, err := os.MkdirTemp("", "mew-capsule-quarantine-*")
	if err != nil {
		return nil, apperr.Wrap(apperr.IO, "capsule.restore", opts.ArchivePath, err)
	}
	defer func() { _ = os.RemoveAll(quarantineDir) }()

	tr := tar.NewReader(f)
	var manifest *Manifest
	seenBlobs := map[string]struct{}{}
	quarantinePaths := map[string]string{}
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, apperr.Wrap(apperr.IO, "capsule.restore", opts.ArchivePath, err)
		}
		name := filepath.ToSlash(strings.TrimPrefix(hdr.Name, "./"))
		switch {
		case name == manifestArchiveName:
			if hdr.Size > maxManifestBytes {
				return nil, apperr.New(apperr.Integrity, "capsule.restore", manifestArchiveName,
					fmt.Sprintf("manifest exceeds %d bytes", maxManifestBytes))
			}
			data, err := io.ReadAll(io.LimitReader(tr, hdr.Size))
			if err != nil {
				return nil, apperr.Wrap(apperr.IO, "capsule.restore", manifestArchiveName, err)
			}
			if int64(len(data)) != hdr.Size {
				return nil, apperr.New(apperr.Integrity, "capsule.restore", manifestArchiveName, "short manifest read")
			}
			manifest, err = DecodeManifestJSON(data)
			if err != nil {
				return nil, err
			}
			if err := VerifyIntegrity(manifest); err != nil {
				return nil, err
			}
		case strings.HasPrefix(name, "blobs/"):
			ref, err := parseBlobArchivePath(name)
			if err != nil {
				return nil, err
			}
			if manifest == nil {
				return nil, apperr.New(apperr.Integrity, "capsule.restore", name, "manifest must precede blobs")
			}
			if !manifestHasBlob(manifest, ref) {
				return nil, apperr.New(apperr.Integrity, "capsule.restore", name, "unexpected blob in archive")
			}
			key := ref.Algo + "/" + ref.Hex
			if _, ok := seenBlobs[key]; ok {
				return nil, apperr.New(apperr.Integrity, "capsule.restore", name, "duplicate blob entry")
			}
			seenBlobs[key] = struct{}{}
			if hdr.Size > maxBlobBytes {
				return nil, apperr.New(apperr.Integrity, "capsule.restore", name,
					fmt.Sprintf("blob exceeds %d bytes", maxBlobBytes))
			}
			data, err := readVerifiedBlob(io.LimitReader(tr, hdr.Size), ref)
			if err != nil {
				return nil, err
			}
			if int64(len(data)) != hdr.Size {
				return nil, apperr.New(apperr.Integrity, "capsule.restore", name, "short blob read")
			}
			qpath := filepath.Join(quarantineDir, ref.Algo, ref.Hex)
			if err := os.MkdirAll(filepath.Dir(qpath), 0o700); err != nil {
				return nil, apperr.Wrap(apperr.IO, "capsule.restore", name, err)
			}
			if err := os.WriteFile(qpath, data, 0o600); err != nil {
				return nil, apperr.Wrap(apperr.IO, "capsule.restore", name, err)
			}
			quarantinePaths[key] = qpath
		default:
			return nil, apperr.New(apperr.Integrity, "capsule.restore", name, "unexpected archive member")
		}
	}
	if manifest == nil {
		return nil, apperr.New(apperr.Integrity, "capsule.restore", opts.ArchivePath, "missing capsule.json")
	}
	for _, ref := range manifest.Blobs {
		key := ref.Algo + "/" + ref.Hex
		if _, ok := seenBlobs[key]; !ok {
			return nil, apperr.New(apperr.Integrity, "capsule.restore", ref.BlobPath(), "missing blob in archive")
		}
	}
	if trailing, err := io.Copy(io.Discard, f); err != nil {
		return nil, apperr.Wrap(apperr.IO, "capsule.restore", opts.ArchivePath, err)
	} else if trailing > 0 {
		return nil, apperr.New(apperr.Integrity, "capsule.restore", opts.ArchivePath, "trailing data after archive end")
	}
	for _, ref := range manifest.Blobs {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		key := ref.Algo + "/" + ref.Hex
		qpath := quarantinePaths[key]
		blobFile, err := os.Open(qpath)
		if err != nil {
			return nil, apperr.Wrap(apperr.IO, "capsule.restore", ref.BlobPath(), err)
		}
		err = opts.WriteBlob(ctx, ref, blobFile)
		_ = blobFile.Close()
		if err != nil {
			return nil, err
		}
	}
	return manifest, nil
}

func writeTarBytes(tw *tar.Writer, name string, data []byte) error {
	hdr := &tar.Header{
		Name:   name,
		Mode:   0o644,
		Size:   int64(len(data)),
		Format: tar.FormatUSTAR,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return apperr.Wrap(apperr.IO, "capsule.create", name, err)
	}
	if _, err := tw.Write(data); err != nil {
		return apperr.Wrap(apperr.IO, "capsule.create", name, err)
	}
	return nil
}

func writeTarStream(tw *tar.Writer, name string, rc io.Reader) error {
	data, err := io.ReadAll(io.LimitReader(rc, maxBlobBytes+1))
	if err != nil {
		return apperr.Wrap(apperr.IO, "capsule.create", name, err)
	}
	if int64(len(data)) > maxBlobBytes {
		return apperr.New(apperr.Integrity, "capsule.create", name,
			fmt.Sprintf("blob exceeds %d bytes", maxBlobBytes))
	}
	return writeTarBytes(tw, name, data)
}

func readVerifiedBlob(r io.Reader, ref BlobRef) ([]byte, error) {
	limited := &io.LimitedReader{R: r, N: maxBlobBytes + 1}
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, apperr.Wrap(apperr.IO, "capsule.restore", ref.BlobPath(), err)
	}
	if int64(len(data)) > maxBlobBytes {
		return nil, apperr.New(apperr.Integrity, "capsule.restore", ref.BlobPath(),
			fmt.Sprintf("blob exceeds %d bytes", maxBlobBytes))
	}
	if err := verifyBlobDigest(data, ref); err != nil {
		return nil, err
	}
	return data, nil
}

func verifyBlobDigest(data []byte, ref BlobRef) error {
	var got string
	switch ref.Algo {
	case "sha256":
		sum := sha256.Sum256(data)
		got = hex.EncodeToString(sum[:])
	case "sha512":
		sum := sha512.Sum512(data)
		got = hex.EncodeToString(sum[:])
	default:
		return apperr.New(apperr.Integrity, "capsule.restore", ref.BlobPath(), "unsupported digest algorithm")
	}
	if got != ref.Hex {
		return apperr.New(apperr.Integrity, "capsule.restore", ref.BlobPath(),
			fmt.Sprintf("digest mismatch: got %s want %s", got, ref.Hex))
	}
	return nil
}

func parseBlobArchivePath(name string) (BlobRef, error) {
	parts := strings.Split(strings.TrimPrefix(name, "blobs/"), "/")
	if len(parts) != 2 {
		return BlobRef{}, apperr.New(apperr.Integrity, "capsule.restore", name, "invalid blob path")
	}
	ref := BlobRef{Algo: parts[0], Hex: parts[1]}
	if err := contentid.ValidateKey(ref.Algo, ref.Hex); err != nil {
		return BlobRef{}, apperr.Wrap(apperr.Integrity, "capsule.restore", name, err)
	}
	if strings.Contains(name, "..") {
		return BlobRef{}, apperr.New(apperr.Integrity, "capsule.restore", name, "path traversal")
	}
	return ref, nil
}

func manifestHasBlob(m *Manifest, ref BlobRef) bool {
	for _, want := range m.Blobs {
		if want.Algo == ref.Algo && want.Hex == ref.Hex {
			return true
		}
	}
	return false
}

// FormatCreateLine returns human-readable create output.
func FormatCreateLine(path, id string) string {
	return fmt.Sprintf("%s\n%s\n", filepath.Base(path), id)
}
