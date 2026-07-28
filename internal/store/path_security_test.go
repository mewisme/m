package store_test

import (
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
