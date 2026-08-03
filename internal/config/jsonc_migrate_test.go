package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSetRejectsScalarIntermediateParent is the structural-ambiguity guard.
// "network" already holds a string, so creating network.timeout underneath it
// would either clobber the string or emit a duplicate "network" member.
// Neither is a change the user asked for, so the edit must fail loudly.
func TestSetRejectsScalarIntermediateParent(t *testing.T) {
	src := []byte("{\n  \"network\": \"fast\"\n}\n")
	_, err := setJSONCPath(src, "network.timeout", "30s")
	if err == nil {
		t.Fatal("setJSONCPath through a scalar parent succeeded; want an error")
	}
	if !errors.Is(err, ErrScalarParent) {
		t.Errorf("err = %v, want ErrScalarParent", err)
	}
}

// TestSetFileRejectsScalarIntermediateParent proves the guard survives the
// public entry point and leaves the file byte-identical.
func TestSetFileRejectsScalarIntermediateParent(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "m.jsonc")
	original := "{\n  // keep me\n  \"network\": \"fast\"\n}\n"
	if err := os.WriteFile(p, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := SetFile(p, "network.timeout", "30s"); err == nil {
		t.Fatal("SetFile through a scalar parent succeeded; want an error")
	}
	after, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != original {
		t.Errorf("file was modified by a failed edit:\n%s", after)
	}
}

// TestSetCreatesMissingIntermediateObjects confirms the guard did not break
// the legitimate case of creating nested objects that do not exist yet.
func TestSetCreatesMissingIntermediateObjects(t *testing.T) {
	src := []byte("{}\n")
	out, err := setJSONCPath(src, "network.timeout", "30s")
	if err != nil {
		t.Fatalf("setJSONCPath: %v", err)
	}
	parsed, err := ParseJSONC(out)
	if err != nil {
		t.Fatalf("result does not parse: %v\n%s", err, out)
	}
	m := parsed.(map[string]any)
	net, ok := m["network"].(map[string]any)
	if !ok {
		t.Fatalf("network is %T, want object", m["network"])
	}
	if net["timeout"] != "30s" {
		t.Errorf("network.timeout = %v, want 30s", net["timeout"])
	}
}

// TestSetPreservesComments is the whole point of splice editing.
func TestSetPreservesComments(t *testing.T) {
	src := []byte(`{
  // registry for the team mirror
  "registry": "https://old.example",
  /* block comment */
  "offline": false
}
`)
	out, err := setJSONCPath(src, "registry", "https://new.example")
	if err != nil {
		t.Fatalf("setJSONCPath: %v", err)
	}
	got := string(out)
	for _, want := range []string{
		"// registry for the team mirror",
		"/* block comment */",
		`"https://new.example"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output lost %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "old.example") {
		t.Errorf("old value survived:\n%s", got)
	}
}

// TestDetectDuplicateKeys catches the silent last-wins behaviour of
// encoding/json, which makes a duplicated setting look like it works.
func TestDetectDuplicateKeys(t *testing.T) {
	cases := []struct {
		name    string
		src     string
		wantDup string
	}{
		{"top level", `{"registry": "a", "registry": "b"}`, "registry"},
		{"nested", `{"network": {"proxy": "a", "proxy": "b"}}`, "network.proxy"},
		{"inside array", `{"a": [{"k": 1, "k": 2}]}`, "a.k"},
		{"none", `{"registry": "a", "offline": true}`, ""},
		{"same name different objects", `{"a": {"k": 1}, "b": {"k": 2}}`, ""},
		{"comment mentions key", "{\n  // \"registry\": ignored\n  \"registry\": \"a\"\n}", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := DetectDuplicateKeys([]byte(tc.src))
			if tc.wantDup == "" {
				if err != nil {
					t.Fatalf("unexpected duplicate: %v", err)
				}
				return
			}
			var dk *DuplicateKeyError
			if !errors.As(err, &dk) {
				t.Fatalf("err = %v, want DuplicateKeyError", err)
			}
			if dk.Path != tc.wantDup {
				t.Errorf("duplicate path = %q, want %q", dk.Path, tc.wantDup)
			}
		})
	}
}

// TestMigrationPlanIsDeterministic guards against map-iteration order leaking
// into the plan, which would make migrate output vary between runs.
func TestMigrationPlanIsDeterministic(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "m.jsonc")
	body := `{
  "prefer-offline": true,
  "network": {"timeout_ms": 1500},
  "resolve": {"autoInstallPeers": true, "rejectDeprecated": false}
}
`
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	first, err := PlanMigration(p)
	if err != nil {
		t.Fatalf("PlanMigration: %v", err)
	}
	for i := 0; i < 20; i++ {
		again, err := PlanMigration(p)
		if err != nil {
			t.Fatalf("PlanMigration: %v", err)
		}
		if len(again.Steps) != len(first.Steps) {
			t.Fatalf("step count changed between runs: %d vs %d", len(again.Steps), len(first.Steps))
		}
		for j := range first.Steps {
			if again.Steps[j] != first.Steps[j] {
				t.Fatalf("step %d differs between runs: %+v vs %+v", j, again.Steps[j], first.Steps[j])
			}
		}
	}
	if len(first.Steps) != 4 {
		t.Errorf("planned %d steps, want 4: %+v", len(first.Steps), first.Steps)
	}
}

// TestMigratePreservesComments is the behaviour the old implementation
// refused to attempt: it bailed out whenever a comment was present.
func TestMigratePreservesComments(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "m.jsonc")
	body := `{
  // team mirror
  "registry": "https://mirror.example",
  "network": {
    // was milliseconds
    "timeout_ms": 1500
  }
}
`
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	count, err := MigrateFile(p)
	if err != nil {
		t.Fatalf("MigrateFile: %v", err)
	}
	if count != 1 {
		t.Errorf("migrated %d keys, want 1", count)
	}
	out, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	got := string(out)
	if !strings.Contains(got, "// team mirror") {
		t.Errorf("comment lost:\n%s", got)
	}
	parsed, err := ParseJSONC(out)
	if err != nil {
		t.Fatalf("migrated file does not parse: %v\n%s", err, got)
	}
	net := parsed.(map[string]any)["network"].(map[string]any)
	if net["timeout"] != "1.5s" {
		t.Errorf("network.timeout = %v, want 1.5s", net["timeout"])
	}
	if _, still := net["timeout_ms"]; still {
		t.Errorf("legacy key survived:\n%s", got)
	}
}

// TestCheckMigrationMatchesApply keeps the dry run honest: --check must report
// exactly the renames that running the migration performs.
func TestCheckMigrationMatchesApply(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "m.jsonc")
	body := `{"prefer-offline": true, "network": {"timeout_ms": 2000}}`
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	needed, err := CheckMigration(p)
	if err != nil {
		t.Fatalf("CheckMigration: %v", err)
	}
	count, err := MigrateFile(p)
	if err != nil {
		t.Fatalf("MigrateFile: %v", err)
	}
	if count != len(needed) {
		t.Errorf("CheckMigration reported %d renames, MigrateFile applied %d", len(needed), count)
	}
	// A second run is a no-op.
	again, err := MigrateFile(p)
	if err != nil {
		t.Fatalf("second MigrateFile: %v", err)
	}
	if again != 0 {
		t.Errorf("second migration changed %d keys, want 0", again)
	}
}

// TestMigrateRefusesConflictingKeys: applying the rename would drop a value.
func TestMigrateRefusesConflictingKeys(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "m.jsonc")
	original := `{"network": {"timeout_ms": 1000, "timeout": "5s"}}`
	if err := os.WriteFile(p, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := MigrateFile(p); err == nil {
		t.Fatal("MigrateFile with conflicting keys succeeded; want an error")
	}
	after, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != original {
		t.Errorf("file modified despite the error:\n%s", after)
	}
}
