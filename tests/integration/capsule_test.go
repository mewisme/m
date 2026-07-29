package integration_test

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestCapsuleRoundTripNodeModulesHash(t *testing.T) {
	projDir, cfgPath, _ := setupRegistryProject(t, `{
  "name": "basic-export",
  "version": "1.0.0",
  "dependencies": { "pkg-a": "^1.0.0" }
}`)
	if code, out := runM(t, projDir, cfgPath, "install"); code != 0 {
		t.Fatalf("install exit=%d out=%s", code, out)
	}
	before := hashNodeModules(t, filepath.Join(projDir, "node_modules"))

	capsulePath := filepath.Join(projDir, "basic-export.capsule")
	if code, out := runM(t, projDir, cfgPath, "capsule", "create", "--output", capsulePath); code != 0 {
		t.Fatalf("capsule create exit=%d out=%s", code, out)
	}
	if err := os.RemoveAll(filepath.Join(projDir, "node_modules")); err != nil {
		t.Fatal(err)
	}

	if code, out := runM(t, projDir, cfgPath, "capsule", "restore", capsulePath); code != 0 {
		t.Fatalf("capsule restore exit=%d out=%s", code, out)
	}
	after := hashNodeModules(t, filepath.Join(projDir, "node_modules"))
	if after != before {
		t.Fatalf("node_modules hash mismatch\nbefore=%s\nafter=%s", before, after)
	}
}

func hashNodeModules(t *testing.T, root string) string {
	t.Helper()
	h := sha256.New()
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if _, err := h.Write([]byte(rel + "\n")); err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(data)
		if _, err := h.Write(sum[:]); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return hex.EncodeToString(h.Sum(nil))
}
