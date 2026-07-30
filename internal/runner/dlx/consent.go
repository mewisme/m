package dlx

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/fsx"
)

const consentStoreSchemaVersion = 1

// ConsentStore persists approved ConsentKey digests.
type ConsentStore struct {
	SchemaVersion int      `json:"schemaVersion"`
	Approved      []string `json:"approved"`
}

// ConsentStorePath returns the cross-process consent store path.
func ConsentStorePath(cacheRoot string) string {
	return filepath.Join(cacheRoot, "mx", "consent.v1.json")
}

// LoadConsentStore reads the consent store from disk.
func LoadConsentStore(path string) (ConsentStore, error) {
	var empty ConsentStore
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ConsentStore{SchemaVersion: consentStoreSchemaVersion, Approved: []string{}}, nil
		}
		return empty, apperr.Wrap(apperr.IO, "dlx.consent", path, err)
	}
	var doc ConsentStore
	if err := json.Unmarshal(b, &doc); err != nil {
		return empty, apperr.New(apperr.Integrity, "dlx.consent", path, "corrupt consent store")
	}
	if doc.SchemaVersion != consentStoreSchemaVersion {
		return empty, apperr.New(apperr.Integrity, "dlx.consent", path, "unsupported consent schema")
	}
	if doc.Approved == nil {
		doc.Approved = []string{}
	}
	return doc, nil
}

// HasConsent reports whether key is approved in store.
func (s ConsentStore) HasConsent(key ConsentKey) bool {
	d := key.Digest()
	for _, a := range s.Approved {
		if a == d {
			return true
		}
	}
	return false
}

// MergeConsent atomically persists an approved consent key under lock.
func MergeConsent(ctx context.Context, cacheRoot, mxCacheDir string, key ConsentKey) error {
	release, err := AcquireLock(ctx, mxCacheDir, LockConsent, "store")
	if err != nil {
		return err
	}
	defer release()

	path := ConsentStorePath(cacheRoot)
	store, err := LoadConsentStore(path)
	if err != nil {
		return err
	}
	d := key.Digest()
	for _, a := range store.Approved {
		if a == d {
			return nil
		}
	}
	store.Approved = append(store.Approved, d)
	sort.Strings(store.Approved)
	b, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return apperr.Wrap(apperr.Internal, "dlx.consent", path, err)
	}
	b = append(b, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return apperr.Wrap(apperr.IO, "dlx.consent", path, err)
	}
	if err := fsx.WriteAtomic(path, b, 0o600); err != nil {
		return apperr.Wrap(apperr.IO, "dlx.consent", path, err)
	}
	return nil
}
