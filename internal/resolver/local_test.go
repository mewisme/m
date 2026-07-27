package resolver_test

import (
	"context"
	"strings"
	"testing"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/resolver"
)

func TestResolveFilePlaceholder(t *testing.T) {
	eng, _ := testEngine(t)
	root := writeWorkspace(t, map[string]string{
		"package.json": `{
  "name": "root",
  "version": "1.0.0",
  "dependencies": { "local-pkg": "file:./vendor/local-pkg" }
}`,
		"vendor/local-pkg/package.json": `{
  "name": "local-pkg",
  "version": "9.8.7"
}`,
	})
	res, err := eng.Resolve(context.Background(), root, resolver.ResolveOptions{})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, p := range res.Graph.Packages {
		if p.ID.Name == "local-pkg" && p.ID.Version == "9.8.7" {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing local package: %#v", res.Graph.Packages)
	}
	locals, err := resolver.DecodeLocalSources(res.Extensions)
	if err != nil {
		t.Fatal(err)
	}
	if src, ok := locals["local-pkg@9.8.7"]; !ok || src.Protocol != "file" || src.Path != "vendor/local-pkg" {
		t.Fatalf("local extension=%#v", locals)
	}
}

func TestResolveLinkWithoutManifestVersion(t *testing.T) {
	eng, _ := testEngine(t)
	root := writeWorkspace(t, map[string]string{
		"package.json": `{
  "name": "root",
  "version": "1.0.0",
  "dependencies": { "linked": "link:./linked" }
}`,
		"linked/package.json": `{
  "name": "linked"
}`,
	})
	res, err := eng.Resolve(context.Background(), root, resolver.ResolveOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range res.Graph.Packages {
		if p.ID.Name == "linked" && p.ID.Version != "0.0.0" {
			t.Fatalf("link without version got %s", p.ID.Version)
		}
	}
	locals, err := resolver.DecodeLocalSources(res.Extensions)
	if err != nil {
		t.Fatal(err)
	}
	if src, ok := locals["linked@0.0.0"]; !ok || src.Protocol != "link" {
		t.Fatalf("local extension=%#v", locals)
	}
}

func TestResolvePortalPlaceholder(t *testing.T) {
	eng, _ := testEngine(t)
	root := writeWorkspace(t, map[string]string{
		"package.json": `{
  "name": "root",
  "version": "1.0.0",
  "dependencies": { "portal-pkg": "portal:./portal/pkg" }
}`,
		"portal/pkg/package.json": `{
  "name": "portal-pkg",
  "version": "3.0.0"
}`,
	})
	res, err := eng.Resolve(context.Background(), root, resolver.ResolveOptions{})
	if err != nil {
		t.Fatal(err)
	}
	locals, err := resolver.DecodeLocalSources(res.Extensions)
	if err != nil {
		t.Fatal(err)
	}
	if src, ok := locals["portal-pkg@3.0.0"]; !ok || src.Protocol != "portal" {
		t.Fatalf("local extension=%#v", locals)
	}
}

func TestResolveFileEscapesRoot(t *testing.T) {
	eng, _ := testEngine(t)
	root := writeWorkspace(t, map[string]string{
		"package.json": `{
  "name": "root",
  "version": "1.0.0",
  "dependencies": { "bad": "file:../../outside" }
}`,
	})
	_, err := eng.Resolve(context.Background(), root, resolver.ResolveOptions{})
	if err == nil {
		t.Fatal("expected error")
	}
	if apperr.CodeOf(err) != apperr.Resolve {
		t.Fatalf("code=%s err=%v", apperr.CodeOf(err), err)
	}
	if !strings.Contains(err.Error(), "escapes project root") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHasLocalSources(t *testing.T) {
	if resolver.HasLocalSources(nil) {
		t.Fatal("nil extensions should be false")
	}
}
