package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/config"
)

func FuzzLoadConfig(f *testing.F) {
	f.Add([]byte(`{"registry":"https://example.com"}`))
	f.Add([]byte(`{ // comment\n"registry": 1 }`))
	f.Add([]byte(`{"unknown.owned":true}`))
	f.Add([]byte(`not json`))
	f.Fuzz(func(t *testing.T, data []byte) {
		dir := t.TempDir()
		path := filepath.Join(dir, "m.jsonc")
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatal(err)
		}
		_, err := config.Load(t.Context(), config.LoadOptions{
			CWD:         dir,
			ProjectPath: path,
			GlobalPath:  filepath.Join(dir, "missing-global.jsonc"),
			Env:         []string{},
		})
		if err == nil {
			return
		}
		_ = apperr.CodeOf(err)
	})
}

func TestKnownBadConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "m.jsonc")
	if err := os.WriteFile(path, []byte(`{`), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := config.Load(t.Context(), config.LoadOptions{
		CWD:         dir,
		ProjectPath: path,
		GlobalPath:  filepath.Join(dir, "missing-global.jsonc"),
		Env:         []string{},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}
