package apperr_test

import (
	"errors"
	"testing"

	"github.com/mewisme/m/internal/apperr"
)

func TestOperationFailureUnwrapAndIs(t *testing.T) {
	primary := apperr.New(apperr.Network, "fetch", "pkg", "fetch failed")
	cleanup := errors.New("lock release failed")
	err := apperr.JoinCleanup(primary, cleanup)

	if !errors.Is(err, primary) {
		t.Fatal("expected primary in chain")
	}
	if !errors.Is(err, cleanup) {
		t.Fatal("expected cleanup detectable via Is")
	}
	if apperr.CodeOf(err) != apperr.Network {
		t.Fatalf("code=%s", apperr.CodeOf(err))
	}
}

func TestWithCleanupNilPrimary(t *testing.T) {
	cleanup := errors.New("cleanup only")
	err := apperr.WithCleanup(nil, cleanup)
	if !errors.Is(err, cleanup) {
		t.Fatal("expected cleanup error")
	}
}

func TestWithCleanupNilCleanup(t *testing.T) {
	primary := apperr.New(apperr.Install, "link", "", "link failed")
	err := apperr.WithCleanup(primary)
	if !errors.Is(err, primary) {
		t.Fatal("expected primary only")
	}
}
