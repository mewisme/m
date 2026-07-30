package runner_test

import (
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/runner"
)

func TestLookupExactMatch(t *testing.T) {
	scripts := map[string]string{"dev": "vite", "build": "tsc"}
	got, err := runner.Lookup(scripts, "dev")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != "dev" {
		t.Fatalf("got %v, want [dev]", got)
	}
}

func TestLookupMissingScript(t *testing.T) {
	scripts := map[string]string{"dev": "vite"}
	_, err := runner.Lookup(scripts, "start")
	if err == nil {
		t.Fatal("expected error")
	}
	if apperr.CodeOf(err) != apperr.NotFound {
		t.Fatalf("code %q, want %q", apperr.CodeOf(err), apperr.NotFound)
	}
	if !strings.Contains(err.Error(), "Missing script") {
		t.Fatalf("unexpected message: %v", err)
	}
	if !strings.Contains(err.Error(), "dev") {
		t.Fatalf("expected available scripts in message: %v", err)
	}
}

func TestLookupBadRegex(t *testing.T) {
	scripts := map[string]string{"dev": "vite"}
	_, err := runner.Lookup(scripts, "/[unclosed")
	if err == nil {
		t.Fatal("expected error")
	}
	if apperr.CodeOf(err) != apperr.Usage {
		t.Fatalf("code %q, want %q", apperr.CodeOf(err), apperr.Usage)
	}
}

func TestLookupRegexMultiMatchSorted(t *testing.T) {
	scripts := map[string]string{
		"test:unit": "vitest",
		"test:e2e":  "playwright",
		"build":     "tsc",
		"test:lint": "eslint",
	}
	got, err := runner.Lookup(scripts, `/^test:/`)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"test:e2e", "test:lint", "test:unit"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestLookupRegexNoMatch(t *testing.T) {
	scripts := map[string]string{"dev": "vite"}
	_, err := runner.Lookup(scripts, `/^missing:/`)
	if err == nil {
		t.Fatal("expected error")
	}
	if apperr.CodeOf(err) != apperr.NotFound {
		t.Fatalf("code %q, want %q", apperr.CodeOf(err), apperr.NotFound)
	}
}

func TestExpandHooksOrdering(t *testing.T) {
	scripts := map[string]string{
		"predev":  "echo pre",
		"dev":     "vite",
		"postdev": "echo post",
	}
	stages := runner.ExpandHooks(scripts, "dev")
	if len(stages) != 3 {
		t.Fatalf("got %d stages, want 3", len(stages))
	}
	want := []struct {
		event, script string
	}{
		{"predev", "echo pre"},
		{"dev", "vite"},
		{"postdev", "echo post"},
	}
	for i, w := range want {
		if stages[i].Event != w.event || stages[i].Script != w.script {
			t.Fatalf("stage %d: got %+v, want event=%q script=%q", i, stages[i], w.event, w.script)
		}
	}
}

func TestExpandHooksSkipsMissing(t *testing.T) {
	scripts := map[string]string{"dev": "vite"}
	stages := runner.ExpandHooks(scripts, "dev")
	if len(stages) != 1 || stages[0].Event != "dev" {
		t.Fatalf("got %+v, want single dev stage", stages)
	}
}

func TestExpandPlansMultipleScripts(t *testing.T) {
	scripts := map[string]string{
		"test:a": "a",
		"test:b": "b",
	}
	plans := runner.ExpandPlans(scripts, []string{"test:a", "test:b"})
	if len(plans) != 2 {
		t.Fatalf("got %d plans, want 2", len(plans))
	}
	if plans[0].Name != "test:a" || plans[1].Name != "test:b" {
		t.Fatalf("unexpected plan order: %+v", plans)
	}
}
