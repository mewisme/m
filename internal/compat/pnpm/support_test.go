package pnpm_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/compat/pnpm"
)

func TestValidateSupportedPnpmAcceptsV9(t *testing.T) {
	root := moduleRoot(t)
	path := filepath.Join(root, "fixtures", "locks", "pnpm", "v9", "pnpm-lock.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := pnpm.Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	if err := pnpm.ValidateSupportedPnpm(doc); err != nil {
		t.Fatal(err)
	}
}
