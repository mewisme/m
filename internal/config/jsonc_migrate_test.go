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

// ── duplicate detection via ParseJSONC ─────────────────────────

func TestParseJSONCRejectsDuplicateRootKey(t *testing.T) {
	src := []byte(`{"offline": true, "offline": false}`)
	_, err := ParseJSONC(src)
	if err == nil {
		t.Fatal("ParseJSONC accepted duplicate root key")
	}
	var dk *DuplicateKeyError
	if !errors.As(err, &dk) {
		t.Fatalf("err = %v, want DuplicateKeyError", err)
	}
	if dk.Path != "offline" {
		t.Errorf("path = %q, want offline", dk.Path)
	}
}

func TestParseJSONCRejectsDuplicateNestedKey(t *testing.T) {
	src := []byte(`{"resolve": {"auto_install_peers": true, "auto_install_peers": false}}`)
	_, err := ParseJSONC(src)
	if err == nil {
		t.Fatal("ParseJSONC accepted duplicate nested key")
	}
	var dk *DuplicateKeyError
	if !errors.As(err, &dk) {
		t.Fatalf("err = %v, want DuplicateKeyError", err)
	}
	if dk.Path != "resolve.auto_install_peers" {
		t.Errorf("path = %q, want resolve.auto_install_peers", dk.Path)
	}
}

func TestParseJSONCDuplicateSeparatedByComment(t *testing.T) {
	src := []byte("{\n  // first\n  \"offline\": true,\n  // second\n  \"offline\": false\n}")
	_, err := ParseJSONC(src)
	if err == nil {
		t.Fatal("ParseJSONC accepted duplicate key separated by comments")
	}
}

func TestParseJSONCSameKeyDifferentObjectsValid(t *testing.T) {
	src := []byte(`{"a": {"k": 1}, "b": {"k": 2}}`)
	_, err := ParseJSONC(src)
	if err != nil {
		t.Fatalf("ParseJSONC rejected same key in different objects: %v", err)
	}
}

func TestParseJSONCKeyTextInStringNotDuplicate(t *testing.T) {
	src := []byte(`{"key": "offline is a thing", "offline": true}`)
	_, err := ParseJSONC(src)
	if err != nil {
		t.Fatalf("ParseJSONC rejected valid config: %v", err)
	}
}

func TestDuplicateKeyHasLineColumn(t *testing.T) {
	src := []byte("{\n  \"offline\": true,\n  \"offline\": false\n}")
	_, err := ParseJSONC(src)
	if err == nil {
		t.Fatal("ParseJSONC accepted duplicate")
	}
	var dk *DuplicateKeyError
	if !errors.As(err, &dk) {
		t.Fatalf("err = %v, want DuplicateKeyError", err)
	}
	if dk.Line == 0 {
		t.Error("line should be non-zero")
	}
}

// ── JSONC edit safety ──────────────────────────────────────────

func TestUnsetPreservesSiblingComments(t *testing.T) {
	src := []byte("{\n  // keep\n  \"offline\": true,\n  // also keep\n  \"network\": {\"timeout\": \"5s\"}\n}\n")
	out, changed, err := unsetJSONCPath(src, "offline")
	if err != nil {
		t.Fatalf("unsetJSONCPath: %v", err)
	}
	if !changed {
		t.Fatal("expected changed=true")
	}
	got := string(out)
	for _, want := range []string{"// keep", "// also keep", `"timeout": "5s"`} {
		if !strings.Contains(got, want) {
			t.Errorf("comment/value lost: %q\n%s", want, got)
		}
	}
	if strings.Contains(got, `"offline"`) {
		t.Errorf("offline key survived:\n%s", got)
	}
}

func TestUnsetAbsentKeyIsNoOp(t *testing.T) {
	src := []byte(`{"offline": true}`)
	_, changed, err := unsetJSONCPath(src, "nonexistent")
	if err != nil {
		t.Fatalf("unsetJSONCPath: %v", err)
	}
	if changed {
		t.Fatal("expected changed=false for absent key")
	}
}

func TestUnsetPrunesEmptyParent(t *testing.T) {
	src := []byte("{\n  \"network\": {\n    \"timeout\": \"5s\"\n  }\n}\n")
	out, changed, err := unsetJSONCPath(src, "network.timeout")
	if err != nil {
		t.Fatalf("unsetJSONCPath: %v", err)
	}
	if !changed {
		t.Fatal("expected changed=true")
	}
	got := string(out)
	if strings.Contains(got, "network") || strings.Contains(got, "timeout") {
		t.Errorf("empty parent not pruned:\n%s", got)
	}
}

func TestFailedSetLeavesBytesUnchanged(t *testing.T) {
	src := []byte("{\n  \"resolve\": false\n}\n")
	_, err := setJSONCPath(src, "resolve.auto_install_peers", true)
	if err == nil {
		t.Fatal("expected error for scalar parent")
	}
	if string(src) != "{\n  \"resolve\": false\n}\n" {
		t.Errorf("src modified by failed set:\n%s", src)
	}
}

func TestSetRejectsArrayIntermediateParent(t *testing.T) {
	src := []byte(`{"items": [1, 2, 3]}`)
	_, err := setJSONCPath(src, "items.name", "value")
	if err == nil {
		t.Fatal("expected error for array parent")
	}
}

func TestSetExistingScalar(t *testing.T) {
	src := []byte(`{"offline": false}`)
	out, err := setJSONCPath(src, "offline", true)
	if err != nil {
		t.Fatalf("setJSONCPath: %v", err)
	}
	if !strings.Contains(string(out), `"offline": true`) {
		t.Errorf("value not updated:\n%s", out)
	}
}

func TestSetNestedValuePreservesSibling(t *testing.T) {
	src := []byte("{\n  \"network\": {\n    \"proxy\": \"http://old\"\n  }\n}\n")
	out, err := setJSONCPath(src, "network.proxy", "http://new")
	if err != nil {
		t.Fatalf("setJSONCPath: %v", err)
	}
	got := string(out)
	if !strings.Contains(got, `"http://new"`) {
		t.Errorf("new value missing:\n%s", got)
	}
	if strings.Contains(got, "old") {
		t.Errorf("old value survived:\n%s", got)
	}
}

func TestPreserveCRLF(t *testing.T) {
	src := []byte("{\r\n  \"offline\": false\r\n}\r\n")
	out, err := setJSONCPath(src, "offline", true)
	if err != nil {
		t.Fatalf("setJSONCPath: %v", err)
	}
	if !strings.Contains(string(out), "\r\n") {
		t.Errorf("CRLF lost:\n%q", out)
	}
}

func TestPreserveLineComment(t *testing.T) {
	src := []byte("{\n  // line comment\n  \"offline\": false\n}\n")
	out, err := setJSONCPath(src, "offline", true)
	if err != nil {
		t.Fatalf("setJSONCPath: %v", err)
	}
	if !strings.Contains(string(out), "// line comment") {
		t.Errorf("line comment lost:\n%s", out)
	}
}

func TestPreserveBlockComment(t *testing.T) {
	src := []byte("{\n  /* block */\n  \"offline\": false\n}\n")
	out, err := setJSONCPath(src, "offline", true)
	if err != nil {
		t.Fatalf("setJSONCPath: %v", err)
	}
	if !strings.Contains(string(out), "/* block */") {
		t.Errorf("block comment lost:\n%s", out)
	}
}

// ── migration validation ───────────────────────────────────────

func TestMigrateNetworkTimeoutConversion(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "m.jsonc")
	if err := os.WriteFile(p, []byte(`{"network": {"timeout_ms": 1500}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	count, err := MigrateFile(p)
	if err != nil {
		t.Fatalf("MigrateFile: %v", err)
	}
	if count != 1 {
		t.Fatalf("migrated %d keys, want 1", count)
	}
	out, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	parsed, _ := ParseJSONC(out)
	net := parsed.(map[string]any)["network"].(map[string]any)
	if net["timeout"] != "1.5s" {
		t.Errorf("network.timeout = %v, want 1.5s", net["timeout"])
	}
}

func TestMigratePreferOfflineRename(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "m.jsonc")
	if err := os.WriteFile(p, []byte(`{"prefer-offline": true}`), 0o644); err != nil {
		t.Fatal(err)
	}
	count, err := MigrateFile(p)
	if err != nil {
		t.Fatalf("MigrateFile: %v", err)
	}
	if count != 1 {
		t.Fatalf("migrated %d, want 1", count)
	}
	out, _ := os.ReadFile(p)
	parsed, _ := ParseJSONC(out)
	if parsed.(map[string]any)["prefer_offline"] != true {
		t.Errorf("prefer_offline not present:\n%s", out)
	}
}

func TestMigrateAllLegacyMappings(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "m.jsonc")
	body := `{
  "prefer-offline": true,
  "resolve": {
    "autoInstallPeers": true,
    "strictPeerDependencies": false,
    "rejectDeprecated": true,
    "minimumReleaseAge": 72
  },
  "network": {
    "timeout_ms": 3000
  }
}`
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	plan, err := PlanMigration(p)
	if err != nil {
		t.Fatalf("PlanMigration: %v", err)
	}
	if len(plan.Steps) != 6 {
		t.Errorf("planned %d steps, want 6: %+v", len(plan.Steps), plan.Steps)
	}
	count, err := MigrateFile(p)
	if err != nil {
		t.Fatalf("MigrateFile: %v", err)
	}
	if count != 6 {
		t.Errorf("migrated %d, want 6", count)
	}
}

func TestMigrateNetworkTimeoutStringRejected(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "m.jsonc")
	if err := os.WriteFile(p, []byte(`{"network": {"timeout_ms": "abc"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	plan, err := PlanMigration(p)
	if err != nil {
		t.Fatalf("PlanMigration: %v", err)
	}
	if len(plan.Steps) != 0 {
		t.Errorf("expected 0 steps for invalid timeout, got %d", len(plan.Steps))
	}
	if len(plan.Conflicts) == 0 {
		t.Error("expected conflict for invalid timeout type")
	}
}

func TestMigrateNetworkTimeoutNegativeRejected(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "m.jsonc")
	if err := os.WriteFile(p, []byte(`{"network": {"timeout_ms": -100}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	plan, err := PlanMigration(p)
	if err != nil {
		t.Fatalf("PlanMigration: %v", err)
	}
	if len(plan.Steps) != 0 {
		t.Errorf("expected 0 steps, got %d", len(plan.Steps))
	}
	if len(plan.Conflicts) == 0 {
		t.Error("expected conflict for negative timeout")
	}
}

func TestMigrateNetworkTimeoutFloatRejected(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "m.jsonc")
	if err := os.WriteFile(p, []byte(`{"network": {"timeout_ms": 1.5}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	plan, err := PlanMigration(p)
	if err != nil {
		t.Fatalf("PlanMigration: %v", err)
	}
	if len(plan.Steps) != 0 {
		t.Errorf("expected 0 steps for fractional timeout, got %d", len(plan.Steps))
	}
}

func TestCheckAndApplySamePlan(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "m.jsonc")
	body := `{"prefer-offline": true, "network": {"timeout_ms": 2000}}`
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	plan, err := PlanMigration(p)
	if err != nil {
		t.Fatalf("PlanMigration: %v", err)
	}
	needed, err := CheckMigration(p)
	if err != nil {
		t.Fatalf("CheckMigration: %v", err)
	}
	if len(needed) != len(plan.Steps) {
		t.Errorf("CheckMigration reports %d, plan has %d steps", len(needed), len(plan.Steps))
	}
	for _, s := range plan.Steps {
		if needed[s.From] != s.To {
			t.Errorf("mismatch: CheckMigration maps %s -> %s, plan maps %s -> %s",
				s.From, needed[s.From], s.From, s.To)
		}
	}
}

func TestMigrateDuplicateKeysRejected(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "m.jsonc")
	if err := os.WriteFile(p, []byte(`{"prefer-offline": true, "prefer-offline": false}`), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := MigrateFile(p)
	if err == nil {
		t.Fatal("MigrateFile accepted document with duplicate keys")
	}
}

func TestFailedMigrationLeavesOriginal(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "m.jsonc")
	original := `{"network": {"timeout_ms": 1000, "timeout": "5s"}}`
	if err := os.WriteFile(p, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := MigrateFile(p)
	if err == nil {
		t.Fatal("expected error for conflicting legacy/canonical keys")
	}
	after, _ := os.ReadFile(p)
	if string(after) != original {
		t.Errorf("file modified on failure:\n%s", after)
	}
}

func TestDuplicateKeysRejectedBySetFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "m.jsonc")
	dup := `{"offline": true, "offline": false}`
	if err := os.WriteFile(p, []byte(dup), 0o644); err != nil {
		t.Fatal(err)
	}
	err := SetFile(p, "offline", true)
	if err == nil {
		t.Fatal("SetFile accepted duplicate keys in source")
	}
}
