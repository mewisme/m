package store_test

import (
	"runtime"
	"testing"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/store"
)

func TestValidateKeyRejectsTraversal(t *testing.T) {
	_, err := store.PackageKeyFromIntegrity("sha256-../escape")
	if err == nil {
		t.Fatal("expected error")
	}
	if apperr.CodeOf(err) != apperr.Store {
		t.Fatalf("code=%s", apperr.CodeOf(err))
	}
}

func TestValidateKeyRejectsUnsafeBlobKeys(t *testing.T) {
	cases := []store.Key{
		"sha256/../escape",
		"sha256/abc/def",
	}
	if runtime.GOOS == "windows" {
		cases = append(cases,
			"C:/sha256/abc",
			"//server/share/sha256/abc",
		)
	}
	for _, key := range cases {
		if err := store.ValidateKey(key); err == nil {
			t.Fatalf("expected error for %q", key)
		}
	}
}
