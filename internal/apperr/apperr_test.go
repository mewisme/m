package apperr_test

import (
	"context"
	"errors"
	"testing"

	"github.com/mewisme/mew/internal/apperr"
)

func TestEveryCodeHasExit(t *testing.T) {
	for _, c := range apperr.AllCodes() {
		n := apperr.ExitForCode(c)
		if c == apperr.OK && n != 0 {
			t.Errorf("%s: want exit 0, got %d", c, n)
		}
		if c == apperr.Usage && n != 2 {
			t.Errorf("%s: want exit 2, got %d", c, n)
		}
		if c == apperr.Cancelled && n != 130 {
			t.Errorf("%s: want exit 130, got %d", c, n)
		}
		if c != apperr.OK && c != apperr.Usage && c != apperr.Cancelled && n != 1 {
			t.Errorf("%s: want exit 1, got %d", c, n)
		}
	}
}

func TestUnknownCodeExitOne(t *testing.T) {
	if got := apperr.ExitForCode(apperr.Code("ERR_M_UNKNOWN_XYZ")); got != 1 {
		t.Fatalf("got %d", got)
	}
}

func TestWrapUnwrapAndExit(t *testing.T) {
	base := errors.New("boom")
	err := apperr.Wrap(apperr.IO, "read", "m.lock", base)
	if !errors.Is(err, base) {
		t.Fatal("Unwrap broken")
	}
	if apperr.CodeOf(err) != apperr.IO {
		t.Fatalf("code=%s", apperr.CodeOf(err))
	}
	if apperr.ExitCode(err) != 1 {
		t.Fatalf("exit=%d", apperr.ExitCode(err))
	}
}

func TestCancelledExit(t *testing.T) {
	err := apperr.Wrap(apperr.Cancelled, "install", "", context.Canceled)
	if apperr.ExitCode(err) != 130 {
		t.Fatalf("exit=%d", apperr.ExitCode(err))
	}
}

func TestExitHintOverrides(t *testing.T) {
	err := apperr.New(apperr.Internal, "x", "", "y")
	err.ExitHint = 99
	if apperr.ExitCode(err) != 99 {
		t.Fatalf("exit=%d", apperr.ExitCode(err))
	}
}

func TestNilExitZero(t *testing.T) {
	if apperr.ExitCode(nil) != 0 {
		t.Fatal("nil should exit 0")
	}
}
