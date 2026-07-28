package resolver_test

import (
	"testing"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/lockfile"
	"github.com/mewisme/mew/internal/resolver"
)

func TestValidateLocalPathsRejectsEscape(t *testing.T) {
	err := resolver.ValidateLocalPaths(map[string]resolver.LocalSource{
		"evil@1.0.0": {Protocol: "file", Path: "../../outside"},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if apperr.CodeOf(err) != apperr.Lockfile {
		t.Fatalf("code=%s", apperr.CodeOf(err))
	}
}

func TestDecodeLocalSourcesRejectsUntrustedPath(t *testing.T) {
	ext := lockfile.Extensions{
		resolver.LocalExtensionKey: []byte(`{"bad@1.0.0":{"protocol":"file","path":"../escape"}}`),
	}
	_, err := resolver.DecodeLocalSources(ext)
	if err == nil {
		t.Fatal("expected error")
	}
}
