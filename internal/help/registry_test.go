package help_test

import (
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/help"
)

func TestDefaultRegistry(t *testing.T) {
	r, err := help.Default()
	if err != nil {
		t.Fatal(err)
	}
	topics := r.Topics()
	if len(topics) < 10 {
		t.Fatalf("expected curated topics, got %d", len(topics))
	}
	topic, body, err := r.Lookup("runner")
	if err != nil {
		t.Fatal(err)
	}
	if topic.ID != "runner" || len(body) == 0 {
		t.Fatalf("lookup runner: %+v body=%d", topic, len(body))
	}
	if !strings.Contains(string(body), "docs/runner.md") {
		t.Fatalf("missing see-also pointer:\n%s", body)
	}
}

func TestAliasAndErrorLookup(t *testing.T) {
	r, err := help.Default()
	if err != nil {
		t.Fatal(err)
	}
	topic, _, err := r.Lookup("compat")
	if err != nil || topic.ID != "compatibility" {
		t.Fatalf("alias compat: topic=%v err=%v", topic, err)
	}
	topic, _, err = r.LookupError("ERR_M_LOCKFILE")
	if err != nil || topic.ID != "errors/ERR_M_LOCKFILE" {
		t.Fatalf("error lookup: topic=%v err=%v", topic, err)
	}
	topic, _, err = r.ResolveArgs([]string{"errors", "ERR_M_POLICY"})
	if err != nil || topic.ID != "errors/ERR_M_POLICY" {
		t.Fatalf("resolve args: topic=%v err=%v", topic, err)
	}
}

func TestUnknownTopic(t *testing.T) {
	r, err := help.Default()
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = r.Lookup("no-such-topic")
	if !apperr.IsUsage(err) {
		t.Fatalf("want usage error, got %v", err)
	}
}
