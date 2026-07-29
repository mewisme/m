package store

import (
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/contentid"
	"github.com/mewisme/mew/internal/fsx"
)

const maxVerifiedBlobBytes = 512 << 20 // 512 MiB; matches fetch body cap

type parsedKey struct {
	algo string
	hex  string
}

// ValidateKey checks a blob store key for safe layout and supported digest form.
func ValidateKey(key Key) error {
	_, err := parseKey(key)
	return err
}

// PutVerified streams content into the store, verifying the digest matches key.
func (d *Dir) PutVerified(ctx context.Context, key Key, r io.Reader) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if d == nil || d.Root == "" {
		return apperr.New(apperr.IO, "store.put", string(key), "nil store")
	}
	pk, err := parseVerifiedKey(key)
	if err != nil {
		return err
	}
	path := d.BlobPath(key)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return apperr.Wrap(apperr.IO, "store.put", string(key), err)
	}
	data, err := readAndHash(r, pk)
	if err != nil {
		return err
	}
	return writeAtomicVerified(path, pk, data)
}

// GetVerified reads a blob and verifies its digest matches key.
func (d *Dir) GetVerified(ctx context.Context, key Key) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if d == nil || d.Root == "" {
		return nil, apperr.New(apperr.IO, "store.get", string(key), "nil store")
	}
	pk, err := parseVerifiedKey(key)
	if err != nil {
		return nil, err
	}
	path := d.BlobPath(key)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, apperr.Wrap(apperr.NotFound, "store.get", string(key), err)
		}
		return nil, apperr.Wrap(apperr.IO, "store.get", string(key), err)
	}
	if err := verifyBytesDigest(data, pk); err != nil {
		_ = quarantineBlob(d.Root, path, pk)
		return nil, err
	}
	return data, nil
}

// OpenVerified opens a blob for streaming read; Close verifies the digest.
func (d *Dir) OpenVerified(ctx context.Context, key Key) (io.ReadCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if d == nil || d.Root == "" {
		return nil, apperr.New(apperr.IO, "store.open", string(key), "nil store")
	}
	pk, err := parseVerifiedKey(key)
	if err != nil {
		return nil, err
	}
	path := d.BlobPath(key)
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, apperr.Wrap(apperr.NotFound, "store.open", string(key), err)
		}
		return nil, apperr.Wrap(apperr.IO, "store.open", string(key), err)
	}
	h, err := newBlobHasher(pk.algo)
	if err != nil {
		_ = f.Close()
		return nil, err
	}
	return &verifiedReadCloser{
		Reader: io.TeeReader(f, h),
		file:   f,
		h:      h,
		key:    string(key),
		want:   pk.hex,
	}, nil
}

// ExistsVerified reports whether key is present with matching content.
// Corrupt blobs are quarantined and reported as absent.
func (d *Dir) ExistsVerified(key Key) (bool, error) {
	if d == nil || d.Root == "" {
		return false, apperr.New(apperr.IO, "store.exists", string(key), "nil store")
	}
	pk, err := parseVerifiedKey(key)
	if err != nil {
		return false, err
	}
	path := d.BlobPath(key)
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, apperr.Wrap(apperr.IO, "store.exists", string(key), err)
	}
	if err := verifyFileDigest(path, pk); err != nil {
		if apperr.CodeOf(err) == apperr.Integrity {
			_ = quarantineBlob(d.Root, path, pk)
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// VerifyFileDigest checks that path content matches algo/hex.
func VerifyFileDigest(path, algo, wantHex string) error {
	pk, err := parseKey(Key(algo + "/" + wantHex))
	if err != nil {
		return err
	}
	return verifyFileDigest(path, pk)
}

func parseKey(key Key) (parsedKey, error) {
	s := filepath.ToSlash(string(key))
	if err := rejectUnsafeKeyPath(s); err != nil {
		return parsedKey{}, err
	}
	algo, hex, ok := strings.Cut(s, "/")
	if !ok || algo == "" || hex == "" {
		return parsedKey{}, apperr.New(apperr.Store, "store.key", s, "invalid key")
	}
	if err := contentid.ValidateKey(algo, hex); err != nil {
		return parsedKey{}, err
	}
	return parsedKey{algo: algo, hex: hex}, nil
}

func parseVerifiedKey(key Key) (parsedKey, error) {
	pk, err := parseKey(key)
	if err != nil {
		return parsedKey{}, err
	}
	switch pk.algo {
	case "sha256", "sha512":
		return pk, nil
	default:
		return parsedKey{}, apperr.New(apperr.Store, "store.key", string(key), "unsupported algorithm for verified blob")
	}
}

func rejectUnsafeKeyPath(s string) error {
	if s == "" {
		return apperr.New(apperr.Store, "store.key", s, "empty key")
	}
	if strings.Contains(s, "..") {
		return apperr.New(apperr.Store, "store.key", s, "invalid key")
	}
	if len(s) >= 2 && s[1] == ':' {
		return apperr.New(apperr.Store, "store.key", s, "drive path")
	}
	if strings.HasPrefix(s, "//") {
		return apperr.New(apperr.Store, "store.key", s, "unc path")
	}
	if strings.Contains(s, "\\") {
		return apperr.New(apperr.Store, "store.key", s, "invalid key separators")
	}
	if strings.HasPrefix(s, "/") {
		return apperr.New(apperr.Store, "store.key", s, "absolute path")
	}
	return nil
}

func newBlobHasher(algo string) (hash.Hash, error) {
	switch algo {
	case "sha512":
		return sha512.New(), nil
	case "sha256":
		return sha256.New(), nil
	case "sha1":
		return sha1.New(), nil
	default:
		return nil, apperr.New(apperr.Integrity, "store.hash", algo, "unsupported algorithm")
	}
}

func readAndHash(r io.Reader, pk parsedKey) ([]byte, error) {
	h, err := newBlobHasher(pk.algo)
	if err != nil {
		return nil, err
	}
	limited := &io.LimitedReader{R: r, N: maxVerifiedBlobBytes + 1}
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, apperr.Wrap(apperr.IO, "store.put", pk.algo+"/"+pk.hex, err)
	}
	if int64(len(data)) > maxVerifiedBlobBytes {
		return nil, apperr.New(apperr.Integrity, "store.put", pk.algo+"/"+pk.hex,
			fmt.Sprintf("body exceeds %d bytes", maxVerifiedBlobBytes))
	}
	if _, err := h.Write(data); err != nil {
		return nil, apperr.Wrap(apperr.IO, "store.put", pk.algo+"/"+pk.hex, err)
	}
	got := hex.EncodeToString(h.Sum(nil))
	if got != pk.hex {
		return nil, apperr.New(apperr.Integrity, "store.put", pk.algo+"/"+pk.hex,
			fmt.Sprintf("digest mismatch: got %s want %s", got, pk.hex))
	}
	return data, nil
}

func verifyBytesDigest(data []byte, pk parsedKey) error {
	h, err := newBlobHasher(pk.algo)
	if err != nil {
		return err
	}
	if _, err := h.Write(data); err != nil {
		return apperr.Wrap(apperr.IO, "store.verify", pk.algo+"/"+pk.hex, err)
	}
	got := hex.EncodeToString(h.Sum(nil))
	if got != pk.hex {
		return apperr.New(apperr.Integrity, "store.verify", pk.algo+"/"+pk.hex,
			fmt.Sprintf("digest mismatch: got %s want %s", got, pk.hex))
	}
	return nil
}

func verifyFileDigest(path string, pk parsedKey) error {
	f, err := os.Open(path)
	if err != nil {
		return apperr.Wrap(apperr.IO, "store.verify", path, err)
	}
	defer func() { _ = f.Close() }()
	h, err := newBlobHasher(pk.algo)
	if err != nil {
		return err
	}
	limited := &io.LimitedReader{R: f, N: maxVerifiedBlobBytes + 1}
	if _, err := io.Copy(h, limited); err != nil {
		return apperr.Wrap(apperr.IO, "store.verify", path, err)
	}
	if limited.N <= 0 {
		return apperr.New(apperr.Integrity, "store.verify", pk.algo+"/"+pk.hex,
			fmt.Sprintf("body exceeds %d bytes", maxVerifiedBlobBytes))
	}
	got := hex.EncodeToString(h.Sum(nil))
	if got != pk.hex {
		return apperr.New(apperr.Integrity, "store.verify", pk.algo+"/"+pk.hex,
			fmt.Sprintf("digest mismatch: got %s want %s", got, pk.hex))
	}
	return nil
}

func writeAtomic(path string, data []byte) error {
	if err := fsx.PublishFileDurable(path, data, 0o644); err != nil {
		if _, statErr := os.Stat(path); statErr == nil {
			return nil
		}
		return apperr.Wrap(apperr.IO, "store.put", path, err)
	}
	return nil
}

func writeAtomicVerified(path string, pk parsedKey, data []byte) error {
	if err := fsx.PublishFileDurable(path, data, 0o644); err != nil {
		if _, statErr := os.Stat(path); statErr == nil {
			if verr := verifyFileDigest(path, pk); verr != nil {
				return verr
			}
			return nil
		}
		return apperr.Wrap(apperr.IO, "store.put", path, err)
	}
	return nil
}

func quarantineBlob(storeRoot, blobPath string, pk parsedKey) error {
	if blobPath == "" {
		return nil
	}
	if _, err := os.Stat(blobPath); os.IsNotExist(err) {
		return nil
	}
	quarantineRoot := filepath.Join(storeRoot, ".quarantine", pk.algo)
	if err := os.MkdirAll(quarantineRoot, 0o755); err != nil {
		return apperr.Wrap(apperr.Store, "store.quarantine", quarantineRoot, err)
	}
	dest := filepath.Join(quarantineRoot, pk.hex)
	if _, err := os.Stat(dest); err == nil {
		_ = os.RemoveAll(dest)
	}
	if err := os.Rename(blobPath, dest); err != nil {
		return apperr.Wrap(apperr.Store, "store.quarantine", blobPath, err)
	}
	return nil
}

type verifiedReadCloser struct {
	io.Reader
	file io.Closer
	h    hash.Hash
	key  string
	want string
}

func (v *verifiedReadCloser) Close() error {
	var closeErr error
	if v.file != nil {
		closeErr = v.file.Close()
	}
	got := hex.EncodeToString(v.h.Sum(nil))
	if got != v.want {
		err := apperr.New(apperr.Integrity, "store.open", v.key,
			fmt.Sprintf("digest mismatch: got %s want %s", got, v.want))
		if closeErr != nil {
			return closeErr
		}
		return err
	}
	return closeErr
}
