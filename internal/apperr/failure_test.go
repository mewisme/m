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

func TestOperationFailureNestedCleanupCauses(t *testing.T) {
	primary := apperr.New(apperr.Network, "fetch", "pkg", "fetch failed")
	currentErr := apperr.New(apperr.Integrity, "transaction.current", "current.bad", "malformed current generation file")
	lockErr := apperr.New(apperr.Transaction, "transaction.lock", "", "lock release failed")
	err := apperr.JoinCleanup(primary, errors.Join(currentErr, lockErr))

	if !errors.Is(err, primary) {
		t.Fatal("expected primary in chain")
	}
	if !errors.Is(err, currentErr) {
		t.Fatal("expected current cleanup in chain")
	}
	if !errors.Is(err, lockErr) {
		t.Fatal("expected lock cleanup in chain")
	}
	if apperr.CodeOf(err) != apperr.Network {
		t.Fatalf("code=%s", apperr.CodeOf(err))
	}
}
