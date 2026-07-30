package runner_test

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCorpusHoistedLayout(t *testing.T) {
	root := fixturePath(t, "dispatch", "collision-matrix")
	if _, err := os.Stat(filepath.Join(root, "package.json")); err != nil {
		t.Fatal(err)
	}
}

func TestCorpusIsolatedLayout(t *testing.T) {
	root := fixturePath(t, "runner", "basic-scripts")
	if _, err := os.Stat(filepath.Join(root, "package.json")); err != nil {
		t.Fatal(err)
	}
}

func TestCorpusClassicBinLayout(t *testing.T) {
	root := fixturePath(t, "mx", "local-hit")
	if _, err := os.Stat(filepath.Join(root, "package.json")); err != nil {
		t.Fatal(err)
	}
}
