package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// writeCfg writes a config file under dir and returns its path.
func writeCfg(t *testing.T, dir, name, body string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", p, err)
	}
	return p
}

// loadLayered loads with an explicit global and project file and no ambient env.
func loadLayered(t *testing.T, dir, global, project string, env []string) *Effective {
	t.Helper()
	eff, err := Load(t.Context(), LoadOptions{
		CWD:         dir,
		ProjectRoot: dir,
		GlobalPath:  global,
		ProjectPath: project,
		Env:         env,
	})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return eff
}

// TestLayersRetainShadowedValues is the core of the layered model: a value
// overridden by a higher layer must still be readable at its own scope.
func TestLayersRetainShadowedValues(t *testing.T) {
	dir := t.TempDir()
	global := writeCfg(t, dir, "config.jsonc", `{"registry": "https://user.example"}`)
	project := writeCfg(t, dir, "m.jsonc", `{"registry": "https://project.example"}`)

	eff := loadLayered(t, dir, global, project, []string{})

	got, err := Get(eff, "registry")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Raw != "https://project.example" {
		t.Errorf("effective registry = %v, want project value", got.Raw)
	}
	if got.Source != SourceProject {
		t.Errorf("effective source = %v, want %v", got.Source, SourceProject)
	}

	// The user value is shadowed, not lost.
	user, err := GetAtScope(eff, ScopeUser, "registry")
	if err != nil {
		t.Fatalf("GetAtScope(user): %v", err)
	}
	if user.Raw != "https://user.example" {
		t.Errorf("user registry = %v, want the shadowed user value", user.Raw)
	}
	if user.Path != global {
		t.Errorf("user path = %q, want %q", user.Path, global)
	}
}

// TestGetAtScopeNotSet distinguishes "absent at this scope" from "unknown key".
func TestGetAtScopeNotSet(t *testing.T) {
	dir := t.TempDir()
	global := writeCfg(t, dir, "config.jsonc", `{"registry": "https://user.example"}`)
	project := filepath.Join(dir, "m.jsonc") // absent

	eff := loadLayered(t, dir, global, project, []string{})

	if _, err := GetAtScope(eff, ScopeProject, "registry"); !errors.Is(err, ErrNotSet) {
		t.Errorf("GetAtScope(project) err = %v, want ErrNotSet", err)
	}
	// An unknown key is a different failure, not ErrNotSet.
	_, err := GetAtScope(eff, ScopeUser, "no.such.key")
	if err == nil || errors.Is(err, ErrNotSet) {
		t.Errorf("GetAtScope(unknown) err = %v, want a non-ErrNotSet error", err)
	}
}

// TestListAtScopeOnlyReturnsThatScope proves ListAtScope does not fall through
// to defaults or to other files.
func TestListAtScopeOnlyReturnsThatScope(t *testing.T) {
	dir := t.TempDir()
	global := writeCfg(t, dir, "config.jsonc", `{"registry": "https://user.example"}`)
	project := writeCfg(t, dir, "m.jsonc", `{"offline": true}`)

	eff := loadLayered(t, dir, global, project, []string{})

	userEntries := ListAtScope(eff, ScopeUser)
	if len(userEntries) != 1 || userEntries[0].Key != "registry" {
		t.Fatalf("user entries = %+v, want exactly [registry]", userEntries)
	}
	projEntries := ListAtScope(eff, ScopeProject)
	if len(projEntries) != 1 || projEntries[0].Key != "offline" {
		t.Fatalf("project entries = %+v, want exactly [offline]", projEntries)
	}
	// Effective still sees every key including schema defaults.
	if len(List(eff)) <= len(userEntries)+len(projEntries) {
		t.Errorf("effective list should include defaults, got %d entries", len(List(eff)))
	}
}

// TestExplainOrdersLayersAndMarksWinner checks the full resolution chain.
func TestExplainOrdersLayersAndMarksWinner(t *testing.T) {
	dir := t.TempDir()
	global := writeCfg(t, dir, "config.jsonc", `{"registry": "https://user.example"}`)
	project := writeCfg(t, dir, "m.jsonc", `{"registry": "https://project.example"}`)

	eff, err := Load(t.Context(), LoadOptions{
		CWD:         dir,
		ProjectRoot: dir,
		GlobalPath:  global,
		ProjectPath: project,
		Env:         []string{"MEW_REGISTRY=https://env.example"},
	})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	chain, err := Explain(eff, "registry")
	if err != nil {
		t.Fatalf("Explain: %v", err)
	}
	want := []Source{SourceDefaults, SourceGlobal, SourceProject, SourceEnv}
	if len(chain) != len(want) {
		t.Fatalf("chain has %d rungs (%+v), want %d", len(chain), chain, len(want))
	}
	for i, src := range want {
		if chain[i].Source != src {
			t.Errorf("rung %d source = %v, want %v", i, chain[i].Source, src)
		}
	}
	// Env is highest here, so it must be the only rung marked effective.
	marked := 0
	for _, r := range chain {
		if r.Effective {
			marked++
			if r.Source != SourceEnv {
				t.Errorf("effective rung is %v, want %v", r.Source, SourceEnv)
			}
		}
	}
	if marked != 1 {
		t.Errorf("%d rungs marked effective, want exactly 1", marked)
	}
}

// TestExplainMarksSingleWinnerWithDuplicateValues guards the case where two
// layers hold the same value: only the highest may claim to be effective.
func TestExplainMarksSingleWinnerWithDuplicateValues(t *testing.T) {
	dir := t.TempDir()
	global := writeCfg(t, dir, "config.jsonc", `{"registry": "https://same.example"}`)
	project := writeCfg(t, dir, "m.jsonc", `{"registry": "https://same.example"}`)

	eff := loadLayered(t, dir, global, project, []string{})
	chain, err := Explain(eff, "registry")
	if err != nil {
		t.Fatalf("Explain: %v", err)
	}
	marked := 0
	for _, r := range chain {
		if r.Effective {
			marked++
			if r.Source != SourceProject {
				t.Errorf("effective rung is %v, want %v", r.Source, SourceProject)
			}
		}
	}
	if marked != 1 {
		t.Errorf("%d rungs marked effective, want exactly 1", marked)
	}
}

// TestCLILayerWinsOverEverything pins the top of the precedence order.
func TestCLILayerWinsOverEverything(t *testing.T) {
	dir := t.TempDir()
	global := writeCfg(t, dir, "config.jsonc", `{"offline": false}`)
	project := writeCfg(t, dir, "m.jsonc", `{"offline": false}`)

	eff, err := Load(t.Context(), LoadOptions{
		CWD:         dir,
		ProjectRoot: dir,
		GlobalPath:  global,
		ProjectPath: project,
		Env:         []string{"MEW_OFFLINE=0"},
		CLI:         map[string]any{"offline": true},
	})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	v, err := Get(eff, "offline")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if v.Raw != true || v.Source != SourceCLI {
		t.Errorf("offline = %v from %v, want true from cli", v.Raw, v.Source)
	}
}

// TestLegacyAndCanonicalInSameFileConflicts keeps a file from setting the same
// setting twice under two spellings, in either declaration order.
func TestLegacyAndCanonicalInSameFileConflicts(t *testing.T) {
	for _, body := range []string{
		`{"network": {"timeout_ms": 1000, "timeout": "1s"}}`,
		`{"network": {"timeout": "1s", "timeout_ms": 1000}}`,
	} {
		dir := t.TempDir()
		project := writeCfg(t, dir, "m.jsonc", body)
		_, err := Load(t.Context(), LoadOptions{
			CWD:         dir,
			ProjectRoot: dir,
			GlobalPath:  filepath.Join(dir, "absent.jsonc"),
			ProjectPath: project,
			Env:         []string{},
		})
		if err == nil {
			t.Errorf("Load(%s) succeeded, want a conflict error", body)
		}
	}
}
