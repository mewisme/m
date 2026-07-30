package npm

import (
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/apperr"
)

func TestErrMutationUnsupportedMessage(t *testing.T) {
	err := ErrMutationUnsupported("npm.write", "package-lock.json")
	if apperr.CodeOf(err) != apperr.Unsupported {
		t.Fatalf("code=%s", apperr.CodeOf(err))
	}
	msg := err.Error()
	for _, want := range []string{
		"package-lock.json",
		"read-only",
		"to update dependencies:",
		"npm install",
		"to migrate to m.lock:",
		"m lock migrate --from npm --to m --dry-run",
		"m lock migrate --from npm --to m",
	} {
		if strings.Contains(msg, "docs/") {
			t.Fatalf("must not reference docs in binary error: %q", msg)
		}
		if !strings.Contains(msg, want) {
			t.Fatalf("missing %q in %q", want, msg)
		}
	}
}

func TestErrMutationUnsupportedShrinkwrap(t *testing.T) {
	err := ErrMutationUnsupported("npm.write", "npm-shrinkwrap.json")
	if !strings.Contains(err.Error(), "npm-shrinkwrap.json") {
		t.Fatalf("expected shrinkwrap label: %s", err)
	}
}
