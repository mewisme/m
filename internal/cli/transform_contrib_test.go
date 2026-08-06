package cli

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/config"
	"github.com/mewisme/mew/internal/transform"
)

func writeTestFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// emptyEffective returns a minimal Effective config for tests.
func emptyEffective() *config.Effective {
	return &config.Effective{Values: map[string]config.Value{}}
}

// TestBuildTransformContribution_NoTsconfig verifies that when no tsconfig
// is present, the function succeeds with default options.
func TestBuildTransformContribution_NoTsconfig(t *testing.T) {
	dir := t.TempDir()
	entrypoint := filepath.Join(dir, "app.ts")
	writeTestFile(t, dir, "app.ts", "const x = 1;")

	contrib, err := buildTransformContribution(context.Background(), dir, entrypoint, emptyEffective())
	if err != nil {
		t.Fatal(err)
	}
	if contrib == nil {
		t.Fatal("nil contribution")
	}
	if contrib.CleanupHook == nil {
		t.Fatal("nil cleanup hook")
	}
	// Verify env contains default options.
	hasOpts := false
	for _, e := range contrib.ExtraEnv {
		if strings.HasPrefix(e, "MEW_TRANSFORM_OPTIONS=") {
			hasOpts = true
			// Default options should serialize to "{}".
			if !strings.Contains(e, "{}") {
				t.Logf("default options env: %s", e)
			}
		}
	}
	if !hasOpts {
		t.Fatal("missing MEW_TRANSFORM_OPTIONS in extra env")
	}
	// Cleanup must not error.
	if err := contrib.CleanupHook(); err != nil {
		t.Fatalf("cleanup: %v", err)
	}
}

// TestBuildTransformContribution_ValidTsconfig verifies that a valid
// tsconfig is loaded and its options flow into the env.
func TestBuildTransformContribution_ValidTsconfig(t *testing.T) {
	dir := t.TempDir()
	entrypoint := filepath.Join(dir, "src", "app.ts")
	writeTestFile(t, dir, "src/app.ts", "const x = 1;")
	writeTestFile(t, dir, "tsconfig.json", `{"compilerOptions":{"target":"ES2022","module":"ESNext"}}`)

	contrib, err := buildTransformContribution(context.Background(), dir, entrypoint, emptyEffective())
	if err != nil {
		t.Fatal(err)
	}
	if contrib == nil {
		t.Fatal("nil contribution")
	}

	hasOptsDigest := false
	hasTarget := false
	for _, e := range contrib.ExtraEnv {
		if strings.HasPrefix(e, "MEW_TRANSFORM_OPTIONS=") && strings.Contains(e, "ES2022") {
			hasTarget = true
		}
		if strings.HasPrefix(e, "MEW_TRANSFORM_OPTS_DIGEST=") {
			val := strings.TrimPrefix(e, "MEW_TRANSFORM_OPTS_DIGEST=")
			if val != "" {
				hasOptsDigest = true
			}
		}
	}
	if !hasTarget {
		t.Error("MEW_TRANSFORM_OPTIONS missing target ES2022")
	}
	if !hasOptsDigest {
		t.Fatal("missing or empty MEW_TRANSFORM_OPTS_DIGEST")
	}
	if err := contrib.CleanupHook(); err != nil {
		t.Fatalf("cleanup: %v", err)
	}
}

// TestBuildTransformContribution_MalformedJSONC verifies fail-closed behavior:
// a malformed tsconfig causes an error before any session is created.
func TestBuildTransformContribution_MalformedJSONC(t *testing.T) {
	dir := t.TempDir()
	entrypoint := filepath.Join(dir, "app.ts")
	writeTestFile(t, dir, "app.ts", "const x = 1;")
	writeTestFile(t, dir, "tsconfig.json", `{invalid`)

	_, err := buildTransformContribution(context.Background(), dir, entrypoint, emptyEffective())
	if err == nil {
		t.Fatal("expected error for malformed JSONC")
	}

	code := apperr.CodeOf(err)
	if code != apperr.TransformConfigParse {
		t.Fatalf("expected TransformConfigParse, got %s", code)
	}
}

// TestBuildTransformContribution_SessionNotLeakedOnTsconfigError verifies that
// when tsconfig loading fails, no session is created (no port is bound).
func TestBuildTransformContribution_SessionNotLeakedOnTsconfigError(t *testing.T) {
	dir := t.TempDir()
	entrypoint := filepath.Join(dir, "app.ts")
	writeTestFile(t, dir, "app.ts", "const x = 1;")
	writeTestFile(t, dir, "tsconfig.json", `{invalid`)

	contrib, err := buildTransformContribution(context.Background(), dir, entrypoint, emptyEffective())
	if err == nil {
		if contrib != nil && contrib.CleanupHook != nil {
			_ = contrib.CleanupHook()
		}
		t.Fatal("expected error for malformed tsconfig")
	}
	if contrib != nil {
		t.Fatal("contribution must be nil on error")
	}
}

// TestBuildTransformContribution_ErrorCodeStable verifies that known
// tsconfig error kinds map to stable error codes.
func TestBuildTransformContribution_ErrorCodeStable(t *testing.T) {
	tests := []struct {
		name     string
		tsconfig string
		wantCode apperr.Code
	}{
		{
			name:     "malformed JSONC",
			tsconfig: `{invalid`,
			wantCode: apperr.TransformConfigParse,
		},
		{
			name:     "package extends",
			tsconfig: `{"extends":"@scope/tsconfig"}`,
			wantCode: apperr.TransformConfigExtends,
		},
		{
			name:     "extends cycle",
			tsconfig: `{"extends":"./a.json"}`,
			// Note: a.json must also extend the child to form a cycle.
			// For simplicity, test package-style extends which is always unsupported.
			wantCode: apperr.TransformConfigExtends,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfgDir := t.TempDir()
			ep := filepath.Join(cfgDir, "app.ts")
			writeTestFile(t, cfgDir, "app.ts", "const x = 1;")
			writeTestFile(t, cfgDir, "tsconfig.json", tt.tsconfig)
			// For cycle test, also create a.json that extends back.
			if tt.name == "extends cycle" {
				writeTestFile(t, cfgDir, "a.json", `{"extends":"./tsconfig.json"}`)
			}

			_, err := buildTransformContribution(context.Background(), cfgDir, ep, emptyEffective())
			if err == nil {
				t.Fatal("expected error")
			}
			code := apperr.CodeOf(err)
			if code != tt.wantCode {
				t.Fatalf("code=%s, want %s", code, tt.wantCode)
			}
		})
	}
}

// TestBuildTransformContribution_SubjectPathStable verifies that the
// entrypoint path is preserved as the error subject.
func TestBuildTransformContribution_SubjectPathStable(t *testing.T) {
	dir := t.TempDir()
	entrypoint := filepath.Join(dir, "src", "app.ts")
	writeTestFile(t, dir, "src/app.ts", "const x = 1;")
	writeTestFile(t, dir, "tsconfig.json", `{invalid`)

	_, err := buildTransformContribution(context.Background(), dir, entrypoint, emptyEffective())
	if err == nil {
		t.Fatal("expected error")
	}

	var ae *apperr.Error
	if !errors.As(err, &ae) {
		t.Fatalf("expected apperr.Error, got %T: %v", err, err)
	}
	if ae.Subject != entrypoint {
		t.Fatalf("subject=%s, want %s", ae.Subject, entrypoint)
	}
}

// TestBuildTransformContribution_ExtendsCycleError verifies extends cycle
// detection propagates correctly through the contrib layer.
func TestBuildTransformContribution_ExtendsCycleError(t *testing.T) {
	dir := t.TempDir()
	entrypoint := filepath.Join(dir, "app.ts")
	writeTestFile(t, dir, "app.ts", "const x = 1;")
	writeTestFile(t, dir, "tsconfig.json", `{"extends":"./a.json"}`)
	writeTestFile(t, dir, "a.json", `{"extends":"./tsconfig.json"}`)

	_, err := buildTransformContribution(context.Background(), dir, entrypoint, emptyEffective())
	if err == nil {
		t.Fatal("expected cycle error")
	}

	code := apperr.CodeOf(err)
	if code != apperr.TransformConfigExtends {
		t.Fatalf("expected TransformConfigExtends, got %s", code)
	}

	// Verify it's a ConfigError with extends_cycle kind.
	var cfgErr *transform.ConfigError
	if !asConfigError(err, &cfgErr) {
		t.Fatalf("expected ConfigError in chain, got %T: %v", err, err)
	}
	if cfgErr.Kind != transform.ConfigErrExtendsCycle {
		t.Fatalf("expected ConfigErrExtendsCycle, got %s", cfgErr.Kind)
	}
}

// TestBuildTransformContribution_ExtendsDepthOverflowError verifies extends
// depth overflow propagates correctly.
func TestBuildTransformContribution_ExtendsDepthOverflowError(t *testing.T) {
	dir := t.TempDir()
	entrypoint := filepath.Join(dir, "app.ts")
	writeTestFile(t, dir, "app.ts", "const x = 1;")

	// Create chain longer than maxTsconfigDepth (20).
	writeTestFile(t, dir, "base0.json", `{"compilerOptions":{}}`)
	prev := "base0.json"
	for i := 1; i <= 20; i++ {
		name := "cfg" + strings.Repeat("x", i) + ".json"
		writeTestFile(t, dir, name, `{"extends":"./`+prev+`"}`)
		prev = name
	}
	writeTestFile(t, dir, "tsconfig.json", `{"extends":"./`+prev+`"}`)

	_, err := buildTransformContribution(context.Background(), dir, entrypoint, emptyEffective())
	if err == nil {
		t.Fatal("expected depth overflow error")
	}

	code := apperr.CodeOf(err)
	if code != apperr.TransformConfigExtends {
		t.Fatalf("expected TransformConfigExtends, got %s", code)
	}
}

// TestBuildTransformContribution_NonStringExtends verifies non-string extends
// produces the correct error code.
func TestBuildTransformContribution_NonStringExtends(t *testing.T) {
	dir := t.TempDir()
	entrypoint := filepath.Join(dir, "app.ts")
	writeTestFile(t, dir, "app.ts", "const x = 1;")
	writeTestFile(t, dir, "tsconfig.json", `{"extends":42}`)

	_, err := buildTransformContribution(context.Background(), dir, entrypoint, emptyEffective())
	if err == nil {
		t.Fatal("expected error")
	}

	code := apperr.CodeOf(err)
	if code != apperr.TransformConfigExtends {
		t.Fatalf("expected TransformConfigExtends, got %s", code)
	}
}

// TestBuildTransformContribution_EmptyExtends verifies empty-string extends
// produces the correct error code.
func TestBuildTransformContribution_EmptyExtends(t *testing.T) {
	dir := t.TempDir()
	entrypoint := filepath.Join(dir, "app.ts")
	writeTestFile(t, dir, "app.ts", "const x = 1;")
	writeTestFile(t, dir, "tsconfig.json", `{"extends":""}`)

	_, err := buildTransformContribution(context.Background(), dir, entrypoint, emptyEffective())
	if err == nil {
		t.Fatal("expected error")
	}

	code := apperr.CodeOf(err)
	if code != apperr.TransformConfigExtends {
		t.Fatalf("expected TransformConfigExtends, got %s", code)
	}
}

// TestBuildTransformContribution_MissingExtendsTarget verifies a missing
// relative extends file produces the correct error code.
func TestBuildTransformContribution_MissingExtendsTarget(t *testing.T) {
	dir := t.TempDir()
	entrypoint := filepath.Join(dir, "app.ts")
	writeTestFile(t, dir, "app.ts", "const x = 1;")
	writeTestFile(t, dir, "tsconfig.json", `{"extends":"./nonexistent.json"}`)

	_, err := buildTransformContribution(context.Background(), dir, entrypoint, emptyEffective())
	if err == nil {
		t.Fatal("expected error")
	}

	code := apperr.CodeOf(err)
	if code != apperr.TransformConfigExtends {
		t.Fatalf("expected TransformConfigExtends, got %s", code)
	}
}

// TestConfigErrToCodeComplete verifies every ConfigErrorKind maps to a non-Internal code.
func TestConfigErrToCodeComplete(t *testing.T) {
	kinds := []transform.ConfigErrorKind{
		transform.ConfigErrIO,
		transform.ConfigErrParse,
		transform.ConfigErrExtendsMissing,
		transform.ConfigErrExtendsCycle,
		transform.ConfigErrExtendsDepth,
		transform.ConfigErrExtendsPackage,
		transform.ConfigErrExtendsInvalid,
		transform.ConfigErrOptionInvalid,
	}
	for _, k := range kinds {
		code := configErrToCode(k)
		if code == apperr.Internal {
			t.Fatalf("kind %s maps to Internal (unexpected)", k)
		}
	}
}
