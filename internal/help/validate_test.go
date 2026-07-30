package help_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/help"
)

var (
	reMDLink  = regexp.MustCompile(`\[[^\]]+\]\(([^)]+)\)`)
	reExample = regexp.MustCompile("(?m)^```(?:text)?\\n([\\s\\S]*?)```")
)

func TestTopicLinksAndExamples(t *testing.T) {
	r, err := help.Default()
	if err != nil {
		t.Fatal(err)
	}
	codes := map[string]struct{}{}
	for _, c := range apperr.AllCodes() {
		codes[string(c)] = struct{}{}
	}
	for _, topic := range r.Topics() {
		_, body, err := r.Lookup(topic.ID)
		if err != nil {
			t.Fatalf("%s: %v", topic.ID, err)
		}
		text := string(body)
		for _, m := range reMDLink.FindAllStringSubmatch(text, -1) {
			dest := m[1]
			if strings.HasPrefix(dest, "http://") || strings.HasPrefix(dest, "https://") {
				if strings.Contains(strings.ToLower(dest), "token=") || strings.Contains(dest, "@") {
					t.Fatalf("%s: secret-bearing URL %q", topic.ID, dest)
				}
				if !strings.HasPrefix(dest, "https://github.com/mewisme/mew") {
					t.Fatalf("%s: unexpected absolute URL %q", topic.ID, dest)
				}
				continue
			}
			if strings.HasPrefix(dest, "docs/") {
				continue // authoritative doc pointers; curated topics use relative docs paths
			}
			t.Fatalf("%s: unsupported relative link %q", topic.ID, dest)
		}
		for _, block := range reExample.FindAllStringSubmatch(text, -1) {
			for _, line := range strings.Split(block[1], "\n") {
				line = strings.TrimSpace(line)
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				if !strings.HasPrefix(line, "m ") && !strings.HasPrefix(line, "mx ") &&
					!strings.HasPrefix(line, "m\t") && !strings.HasPrefix(line, "mx\t") {
					// Allow bare command forms like `m --help` already covered; table cells are outside fences.
					if strings.HasPrefix(line, "m") || strings.HasPrefix(line, "mx") {
						continue
					}
					t.Fatalf("%s: example line must start with m/mx: %q", topic.ID, line)
				}
			}
		}
		if strings.HasPrefix(topic.ID, "errors/ERR_M_") {
			code := strings.TrimPrefix(topic.ID, "errors/")
			if _, ok := codes[code]; !ok {
				t.Fatalf("topic %s references unknown apperr code", topic.ID)
			}
		}
		for _, rel := range topic.Related {
			if _, _, err := r.Lookup(rel); err != nil {
				t.Fatalf("%s related %q: %v", topic.ID, rel, err)
			}
		}
	}
}

func TestErrorHintTopicsExist(t *testing.T) {
	r, err := help.Default()
	if err != nil {
		t.Fatal(err)
	}
	for _, code := range []string{
		"ERR_M_LOCKFILE", "ERR_M_POLICY", "ERR_M_INTEGRITY",
		"ERR_M_TRANSACTION", "ERR_M_USAGE", "ERR_M_CANCELLED",
	} {
		if _, _, err := r.LookupError(code); err != nil {
			t.Fatalf("%s: %v", code, err)
		}
	}
}
