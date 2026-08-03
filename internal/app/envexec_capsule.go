package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/capsule"
	"github.com/mewisme/mew/internal/lockfile/mlock"
	"github.com/mewisme/mew/internal/runner/envexec"
	"github.com/mewisme/mew/internal/snapshot"
)

func openCapsuleForExec(ctx context.Context, path string) (envexec.CapsuleOpenResult, error) {
	var empty envexec.CapsuleOpenResult
	if path == "" {
		return empty, apperr.New(apperr.Usage, "app.capsule.exec", "", "empty capsule path")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return empty, err
	}
	if _, err := os.Stat(abs); err != nil {
		return empty, apperr.Wrap(apperr.NotFound, "app.capsule.exec", abs, err)
	}
	sum, err := capsuleFileSHA256(abs)
	if err != nil {
		return empty, err
	}
	quarantine, err := os.MkdirTemp("", "mew-capsule-exec-*")
	if err != nil {
		return empty, apperr.Wrap(apperr.IO, "app.capsule.exec", abs, err)
	}
	blobs := map[string]string{}
	man, err := capsule.Restore(ctx, capsule.RestoreOptions{
		ArchivePath: abs,
		WriteBlob: func(ctx context.Context, ref capsule.BlobRef, r io.Reader) error {
			dest := filepath.Join(quarantine, ref.Algo, ref.Hex)
			if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
				return err
			}
			f, err := os.Create(dest)
			if err != nil {
				return err
			}
			_, copyErr := io.Copy(f, r)
			closeErr := f.Close()
			if copyErr != nil {
				return copyErr
			}
			return closeErr
		},
	})
	if err != nil {
		_ = os.RemoveAll(quarantine) // intentional: best-effort cleanup of a partial unpack; the unpack error below is authoritative
		return empty, err
	}
	for _, ref := range man.Blobs {
		key := ref.Algo + "/" + ref.Hex
		blobs[key] = filepath.Join(quarantine, ref.Algo, ref.Hex)
	}
	lockDoc, err := mlock.Decode(man.Lock)
	if err != nil {
		return empty, err
	}
	g, err := mlock.ToGraph(lockDoc)
	if err != nil {
		return empty, err
	}
	graphDigest, err := snapshot.GraphDigest(g)
	if err != nil {
		return empty, err
	}
	_ = graphDigest
	return envexec.CapsuleOpenResult{
		Path:          abs,
		ArchiveDigest: sum,
		Manifest:      man,
		Graph:         g,
		Lock:          man.Lock,
		PackageJSON:   man.Manifest,
		Blobs:         blobs,
	}, nil
}

func capsuleFileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", apperr.Wrap(apperr.IO, "app.capsule.exec", path, err)
	}
	defer func() { _ = f.Close() }()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", apperr.Wrap(apperr.IO, "app.capsule.exec", path, err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
