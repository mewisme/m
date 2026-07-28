package semver_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/semver"
	"github.com/mewisme/mew/internal/testkit"
)

type corpusCase struct {
	Version string `json:"version"`
	Range   string `json:"range"`
	Want    bool   `json:"want"`
	Valid   bool   `json:"valid"`
}

func TestConformanceCorpus(t *testing.T) {
	root := testkit.ModuleRoot(t)
	data, err := os.ReadFile(filepath.Join(root, "testdata", "semver", "corpus.json"))
	if err != nil {
		t.Fatal(err)
	}
	var cases []corpusCase
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	for i, tc := range cases {
		got, err := semver.Satisfies(tc.Version, tc.Range)
		if tc.Valid {
			if err != nil {
				t.Fatalf("case %d %q %q: %v", i, tc.Version, tc.Range, err)
			}
			if got != tc.Want {
				t.Fatalf("case %d %q satisfies %q: got %v want %v", i, tc.Version, tc.Range, got, tc.Want)
			}
		} else if err == nil {
			t.Fatalf("case %d %q %q: expected error", i, tc.Version, tc.Range)
		}
	}
}
