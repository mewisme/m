package capsule

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/contentid"
)

const manifestArchiveName = "capsule.json"

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
	if err := os.MkdirAll(filepath.Dir(opts.OutputPath), 0o755); err != nil {
		return apperr.Wrap(apperr.IO, "capsule.create", opts.OutputPath, err)
	}
	f, err := os.Create(opts.OutputPath)
	if err != nil {
		return apperr.Wrap(apperr.IO, "capsule.create", opts.OutputPath, err)
	}
	defer func() { _ = f.Close() }()
	tw := tar.NewWriter(f)
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
	if err := f.Close(); err != nil {
		return apperr.Wrap(apperr.IO, "capsule.create", opts.OutputPath, err)
	}
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
	tr := tar.NewReader(f)
	var manifest *Manifest
	seenBlobs := map[string]struct{}{}
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
			data, err := io.ReadAll(io.LimitReader(tr, 32<<20))
			if err != nil {
				return nil, apperr.Wrap(apperr.IO, "capsule.restore", manifestArchiveName, err)
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
			if err := opts.WriteBlob(ctx, ref, tr); err != nil {
				return nil, err
			}
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
	data, err := io.ReadAll(io.LimitReader(rc, 512<<20))
	if err != nil {
		return apperr.Wrap(apperr.IO, "capsule.create", name, err)
	}
	return writeTarBytes(tw, name, data)
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
