package process_test

import (
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/process"
)

func TestStripGitWorktreeEnv(t *testing.T) {
	in := []string{
		"PATH=/bin",
		"GIT_DIR=/repo/.git",
		"GIT_WORK_TREE=/repo",
		"GIT_INDEX_FILE=/repo/.git/index",
		"HOME=/tmp/home",
	}
	out := process.StripGitWorktreeEnv(in)
	if len(out) != 2 {
		t.Fatalf("got %d env entries: %v", len(out), out)
	}
	for _, kv := range out {
		if strings.HasPrefix(strings.ToUpper(kv), "GIT_") {
			t.Fatalf("git metadata not stripped: %q", kv)
		}
	}
}
