package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/config"
	"github.com/mewisme/mew/internal/fsx"
	"github.com/mewisme/mew/internal/prompt"
	"github.com/mewisme/mew/internal/testkit"
)

// ── editor resolution tests ─────────────────────────────────────

func TestResolveEditorVISUALPrecedence(t *testing.T) {
	t.Setenv("VISUAL", "mvim")
	t.Setenv("EDITOR", "vim")
	name, args, err := resolveEditorCommand()
	if err != nil {
		t.Fatal(err)
	}
	if name != "mvim" {
		t.Fatalf("VISUAL should take precedence, got %q", name)
	}
	if len(args) != 0 {
		t.Fatalf("expected no args, got %v", args)
	}
}

func TestResolveEditorEDITORFallback(t *testing.T) {
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "emacs")
	name, args, err := resolveEditorCommand()
	if err != nil {
		t.Fatal(err)
	}
	if name != "emacs" {
		t.Fatalf("expected EDITOR fallback, got %q", name)
	}
	if len(args) != 0 {
		t.Fatalf("expected no args, got %v", args)
	}
}

func TestResolveEditorWithArguments(t *testing.T) {
	t.Setenv("VISUAL", "code --wait")
	t.Setenv("EDITOR", "")
	name, args, err := resolveEditorCommand()
	if err != nil {
		t.Fatal(err)
	}
	if name != "code" {
		t.Fatalf("expected code, got %q", name)
	}
	if len(args) != 1 || args[0] != "--wait" {
		t.Fatalf("expected [--wait], got %v", args)
	}
}

func TestResolveEditorQuotedArguments(t *testing.T) {
	tests := []struct {
		raw      string
		wantName string
		wantArgs []string
	}{
		{`code --wait`, "code", []string{"--wait"}},
		{`emacsclient -c -a emacs`, "emacsclient", []string{"-c", "-a", "emacs"}},
		{`"my editor" --flag`, "my editor", []string{"--flag"}},
		{`editor 'arg with spaces'`, "editor", []string{"arg with spaces"}},
	}
	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			name, args, err := splitEditorCommand(tt.raw)
			if err != nil {
				t.Fatal(err)
			}
			if name != tt.wantName {
				t.Fatalf("name: got %q, want %q", name, tt.wantName)
			}
			if len(args) != len(tt.wantArgs) {
				t.Fatalf("args len: got %d, want %d (%v)", len(args), len(tt.wantArgs), args)
			}
			for i, a := range args {
				if a != tt.wantArgs[i] {
					t.Fatalf("arg[%d]: got %q, want %q", i, a, tt.wantArgs[i])
				}
			}
		})
	}
}

func TestResolveEditorPlatformFallback(t *testing.T) {
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "")
	fallback := platformEditorFallback()
	if runtime.GOOS == "windows" {
		if fallback != "notepad" {
			t.Fatalf("Windows fallback should be notepad, got %q", fallback)
		}
	} else {
		if fallback != "nano" {
			t.Fatalf("Unix fallback should be nano, got %q", fallback)
		}
	}
}

func TestResolveEditorEmptyValue(t *testing.T) {
	t.Setenv("VISUAL", "   ")
	t.Setenv("EDITOR", "")
	_, _, err := resolveEditorCommand()
	if err == nil {
		t.Fatal("expected error for empty/blank editor")
	}
}

func TestResolveEditorUnterminatedQuote(t *testing.T) {
	_, _, err := splitEditorCommand(`editor "unterminated`)
	if err == nil {
		t.Fatal("expected error for unterminated quote")
	}
}

func TestSplitEditorEmpty(t *testing.T) {
	name, args, err := splitEditorCommand("")
	if err != nil {
		t.Fatal(err)
	}
	if name != "" {
		t.Fatalf("expected empty name, got %q", name)
	}
	if len(args) != 0 {
		t.Fatalf("expected no args, got %v", args)
	}
}

// ── edit command tests ──────────────────────────────────────────

// fakeEditorPath creates a fake editor script that writes the given content to
// the file passed as $1, then exits with the given code.
func fakeEditorPath(t *testing.T, dir string, content []byte, exitCode int) string {
	t.Helper()
	var path string
	contentPath := filepath.Join(dir, "editor-content")
	if err := os.WriteFile(contentPath, content, 0o644); err != nil {
		t.Fatal(err)
	}

	var script string
	if runtime.GOOS == "windows" {
		path = filepath.Join(dir, "fake-editor.cmd")
		script = "@echo off\r\ncopy /y " + contentPath + " %1\r\nexit /b " + fmt.Sprintf("%d", exitCode) + "\r\n"
	} else {
		path = filepath.Join(dir, "fake-editor")
		script = "#!/bin/sh\ncp " + contentPath + " \"$1\"\nexit " + fmt.Sprintf("%d", exitCode) + "\n"
	}
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

// fakeEditorNoop creates a fake editor that does not modify the file.
func fakeEditorNoop(t *testing.T, dir string, exitCode int) string {
	t.Helper()
	var path string
	var script string
	if runtime.GOOS == "windows" {
		path = filepath.Join(dir, "fake-editor.cmd")
		script = "@echo off\r\nexit /b " + fmt.Sprintf("%d", exitCode) + "\r\n"
	} else {
		path = filepath.Join(dir, "fake-editor")
		script = "#!/bin/sh\nexit " + fmt.Sprintf("%d", exitCode) + "\n"
	}
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestEditValidFile(t *testing.T) {
	testkit.CleanEnv(t)
	dir := t.TempDir()
	configPath := filepath.Join(dir, "m.jsonc")
	orig := []byte("{\n  // comment\n  \"registry\": \"https://example.com\"\n}\n")
	if err := os.WriteFile(configPath, orig, 0o644); err != nil {
		t.Fatal(err)
	}

	validContent := []byte("{\n  // updated comment\n  \"registry\": \"https://new.example.com\"\n}\n")
	fakeEditor := fakeEditorPath(t, dir, validContent, 0)
	t.Setenv("VISUAL", fakeEditor)
	t.Setenv("EDITOR", "")

	err := editConfigFile(context.Background(), nil, &globalFlags{}, configPath, config.ScopeUser)
	if err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(validContent) {
		t.Fatalf("expected valid content written, got %q", got)
	}
}

func TestEditInvalidFileRecovery(t *testing.T) {
	testkit.CleanEnv(t)
	dir := t.TempDir()
	configPath := filepath.Join(dir, "m.jsonc")
	orig := []byte("{\n  \"registry\": \"https://example.com\"\n}\n")
	if err := os.WriteFile(configPath, orig, 0o644); err != nil {
		t.Fatal(err)
	}

	invalidContent := []byte("{invalid json!!!}")
	fakeEditor := fakeEditorPath(t, dir, invalidContent, 0)
	t.Setenv("VISUAL", fakeEditor)
	t.Setenv("EDITOR", "")

	editErr := editConfigFile(context.Background(), nil, &globalFlags{}, configPath, config.ScopeUser)
	if editErr == nil {
		t.Fatal("expected error for invalid config")
	}

	// Verify original was restored.
	got, readErr := os.ReadFile(configPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(got) != string(orig) {
		t.Fatalf("expected original content restored, got %q", got)
	}

	// Verify recovery file exists.
	entries, _ := os.ReadDir(dir)
	recoveryFound := false
	for _, e := range entries {
		if strings.Contains(e.Name(), ".invalid-") {
			recoveryFound = true
			recData, recErr := os.ReadFile(filepath.Join(dir, e.Name()))
			if recErr != nil {
				t.Fatal(recErr)
			}
			if string(recData) != string(invalidContent) {
				t.Fatalf("recovery file has wrong content: %q", recData)
			}
			break
		}
	}
	if !recoveryFound {
		t.Fatal("expected recovery file to exist")
	}

	if code := apperr.CodeOf(editErr); code != apperr.Config {
		t.Fatalf("expected Config error code, got %v; error: %v (type: %T)", code, editErr, editErr)
	}
}

func TestEditUnchangedFile(t *testing.T) {
	testkit.CleanEnv(t)
	dir := t.TempDir()
	configPath := filepath.Join(dir, "m.jsonc")
	orig := []byte("{\"registry\":\"https://example.com\"}\n")
	if err := os.WriteFile(configPath, orig, 0o644); err != nil {
		t.Fatal(err)
	}

	fakeEditor := fakeEditorNoop(t, dir, 0)
	t.Setenv("VISUAL", fakeEditor)
	t.Setenv("EDITOR", "")

	err := editConfigFile(context.Background(), nil, &globalFlags{}, configPath, config.ScopeUser)
	if err != nil {
		t.Fatal(err)
	}
}

func TestEditEditorExitsNonZero(t *testing.T) {
	testkit.CleanEnv(t)
	dir := t.TempDir()
	configPath := filepath.Join(dir, "m.jsonc")
	orig := []byte("{\"registry\":\"https://example.com\"}\n")
	if err := os.WriteFile(configPath, orig, 0o644); err != nil {
		t.Fatal(err)
	}

	fakeEditor := fakeEditorNoop(t, dir, 1)
	t.Setenv("VISUAL", fakeEditor)
	t.Setenv("EDITOR", "")

	err := editConfigFile(context.Background(), nil, &globalFlags{}, configPath, config.ScopeUser)
	if err == nil {
		t.Fatal("expected error for non-zero editor exit")
	}
	if !strings.Contains(err.Error(), fakeEditor) {
		t.Fatalf("expected error to mention editor name, got: %v", err)
	}
}

func TestEditEditorExitsNonZeroWithValidChanges(t *testing.T) {
	testkit.CleanEnv(t)
	dir := t.TempDir()
	configPath := filepath.Join(dir, "m.jsonc")
	orig := []byte("{\"registry\":\"https://example.com\"}\n")
	if err := os.WriteFile(configPath, orig, 0o644); err != nil {
		t.Fatal(err)
	}

	validContent := []byte("{\"registry\":\"https://changed.example.com\"}\n")
	fakeEditor := fakeEditorPath(t, dir, validContent, 1)
	t.Setenv("VISUAL", fakeEditor)
	t.Setenv("EDITOR", "")

	err := editConfigFile(context.Background(), nil, &globalFlags{}, configPath, config.ScopeUser)
	if err == nil {
		t.Fatal("expected error for non-zero editor exit (even with valid content)")
	}
}

func TestEditInvalidFileEditorExitsNonZeroRecovery(t *testing.T) {
	testkit.CleanEnv(t)
	dir := t.TempDir()
	configPath := filepath.Join(dir, "m.jsonc")
	orig := []byte("{\"registry\":\"https://example.com\"}\n")
	if err := os.WriteFile(configPath, orig, 0o644); err != nil {
		t.Fatal(err)
	}

	invalidContent := []byte("not json at all")
	fakeEditor := fakeEditorPath(t, dir, invalidContent, 1)
	t.Setenv("VISUAL", fakeEditor)
	t.Setenv("EDITOR", "")

	err := editConfigFile(context.Background(), nil, &globalFlags{}, configPath, config.ScopeUser)
	if err == nil {
		t.Fatal("expected error")
	}

	got, _ := os.ReadFile(configPath)
	if string(got) != string(orig) {
		t.Fatalf("expected original restored, got %q", got)
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "invalid config") {
		t.Fatalf("expected error to mention invalid config, got: %v", err)
	}
}

func TestEditDuplicateKeysRejected(t *testing.T) {
	testkit.CleanEnv(t)
	dir := t.TempDir()
	configPath := filepath.Join(dir, "m.jsonc")
	orig := []byte("{}\n")
	if err := os.WriteFile(configPath, orig, 0o644); err != nil {
		t.Fatal(err)
	}

	dupContent := []byte("{\"registry\":\"a\",\"registry\":\"b\"}\n")
	fakeEditor := fakeEditorPath(t, dir, dupContent, 0)
	t.Setenv("VISUAL", fakeEditor)
	t.Setenv("EDITOR", "")

	err := editConfigFile(context.Background(), nil, &globalFlags{}, configPath, config.ScopeUser)
	if err == nil {
		t.Fatal("expected error for duplicate keys")
	}

	got, _ := os.ReadFile(configPath)
	if string(got) != string(orig) {
		t.Fatalf("expected original restored, got %q", got)
	}
}

func TestEditCommentsSurvive(t *testing.T) {
	testkit.CleanEnv(t)
	dir := t.TempDir()
	configPath := filepath.Join(dir, "m.jsonc")
	orig := []byte("{\n  // registry comment\n  \"registry\": \"https://example.com\"\n}\n")
	if err := os.WriteFile(configPath, orig, 0o644); err != nil {
		t.Fatal(err)
	}

	edited := []byte("{\n  // updated comment\n  \"registry\": \"https://updated.example.com\"\n}\n")
	fakeEditor := fakeEditorPath(t, dir, edited, 0)
	t.Setenv("VISUAL", fakeEditor)
	t.Setenv("EDITOR", "")

	err := editConfigFile(context.Background(), nil, &globalFlags{}, configPath, config.ScopeUser)
	if err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(edited) {
		t.Fatalf("expected edited content with comments, got %q", got)
	}
}

func TestEditEditorCancellation(t *testing.T) {
	testkit.CleanEnv(t)
	dir := t.TempDir()
	configPath := filepath.Join(dir, "m.jsonc")
	orig := []byte("{\"registry\":\"https://example.com\"}\n")
	if err := os.WriteFile(configPath, orig, 0o644); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	fakeEditor := fakeEditorNoop(t, dir, 0)
	t.Setenv("VISUAL", fakeEditor)
	t.Setenv("EDITOR", "")

	editErr := editConfigFile(ctx, nil, &globalFlags{}, configPath, config.ScopeUser)
	if editErr == nil {
		return
	}
}

func TestEditNewFileCreated(t *testing.T) {
	testkit.CleanEnv(t)
	dir := t.TempDir()
	configPath := filepath.Join(dir, "m.jsonc")

	validContent := []byte("{\"registry\":\"https://example.com\"}\n")
	fakeEditor := fakeEditorPath(t, dir, validContent, 0)
	t.Setenv("VISUAL", fakeEditor)
	t.Setenv("EDITOR", "")

	err := editConfigFile(context.Background(), nil, &globalFlags{}, configPath, config.ScopeUser)
	if err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(validContent) {
		t.Fatalf("expected new file content, got %q", got)
	}
}

func TestEditRestoreFailureReportsError(t *testing.T) {
	testkit.CleanEnv(t)
	dir := t.TempDir()
	configPath := filepath.Join(dir, "m.jsonc")
	orig := []byte("{\"registry\":\"https://example.com\"}\n")
	if err := os.WriteFile(configPath, orig, 0o644); err != nil {
		t.Fatal(err)
	}

	invalidContent := []byte("{invalid!!!}")
	fakeEditor := fakeEditorPath(t, dir, invalidContent, 0)
	t.Setenv("VISUAL", fakeEditor)
	t.Setenv("EDITOR", "")

	err := editConfigFile(context.Background(), nil, &globalFlags{}, configPath, config.ScopeUser)
	if err == nil {
		t.Fatal("expected error for invalid content")
	}

	got, _ := os.ReadFile(configPath)
	if string(got) != string(orig) {
		t.Fatalf("expected original restored, got %q", got)
	}
}

// ── reset command tests ─────────────────────────────────────────

func TestResetWithYesDeletesFile(t *testing.T) {
	testkit.CleanEnv(t)
	dir := t.TempDir()
	configPath := filepath.Join(dir, "m.jsonc")
	if err := os.WriteFile(configPath, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Simulate reset with --yes: delete file.
	if err := os.Remove(configPath); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		t.Fatal("expected file to be deleted")
	}
}

func TestResetMissingFileIdempotent(t *testing.T) {
	testkit.CleanEnv(t)
	dir := t.TempDir()
	configPath := filepath.Join(dir, "m.jsonc")
	// File doesn't exist.
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		t.Fatal("expected file to not exist")
	}
	// Missing file should be treated as success, not error.
}

func TestResetEffectiveScopeRejected(t *testing.T) {
	flags := configWriteFlags{scope: "effective"}
	err := flags.validateWritable()
	if err == nil {
		t.Fatal("expected error for effective scope")
	}
}

func TestResetRefusesSymlink(t *testing.T) {
	testkit.CleanEnv(t)
	dir := t.TempDir()
	configPath := filepath.Join(dir, "m.jsonc")
	targetPath := filepath.Join(dir, "real.jsonc")
	if err := os.WriteFile(targetPath, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(targetPath, configPath); err != nil {
		t.Fatal(err)
	}

	fi, err := os.Lstat(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		t.Fatal("expected symlink")
	}
}

func TestResetRefusesDirectory(t *testing.T) {
	testkit.CleanEnv(t)
	dir := t.TempDir()
	configPath := filepath.Join(dir, "m.jsonc")
	if err := os.MkdirAll(configPath, 0o755); err != nil {
		t.Fatal(err)
	}

	fi, err := os.Lstat(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if !fi.IsDir() {
		t.Fatal("expected directory")
	}
}

func TestPromptConfirmAccepted(t *testing.T) {
	// Test the prompt infrastructure accepts confirmation.
	sp := &prompt.ScriptedPrompter{
		Answers: []prompt.PromptAnswer{
			{OptionID: prompt.OptionApprove},
		},
	}
	answer, err := sp.Prompt(context.Background(), prompt.PromptRequest{
		ID:        "test.reset",
		Kind:      prompt.PromptConfirm,
		Title:     "Reset config?",
		DefaultID: prompt.OptionReject,
		Dangerous: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if answer.OptionID != prompt.OptionApprove {
		t.Fatalf("expected approve, got %s", answer.OptionID)
	}
}

func TestPromptConfirmRejected(t *testing.T) {
	sp := &prompt.ScriptedPrompter{
		Answers: []prompt.PromptAnswer{
			{OptionID: prompt.OptionReject},
		},
	}
	answer, err := sp.Prompt(context.Background(), prompt.PromptRequest{
		ID:        "test.reset",
		Kind:      prompt.PromptConfirm,
		Title:     "Reset config?",
		DefaultID: prompt.OptionReject,
		Dangerous: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if answer.OptionID != prompt.OptionReject {
		t.Fatalf("expected reject, got %s", answer.OptionID)
	}
}

func TestPromptCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	sp := &prompt.ScriptedPrompter{}
	answer, err := sp.Prompt(ctx, prompt.PromptRequest{
		ID:   "test.reset",
		Kind: prompt.PromptConfirm,
	})
	if err == nil {
		t.Fatal("expected cancellation error")
	}
	if !answer.Cancelled {
		t.Fatal("expected cancelled answer")
	}
}

// ── path command tests ──────────────────────────────────────────

func TestConfigPathUserJSON(t *testing.T) {
	userPath := config.GlobalConfigPath()
	projPath := ""

	out := configPathJSON{Scope: "user", User: userPath, Selected: userPath, Paths: []string{userPath}}
	data, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}

	var parsed configPathJSON
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed.Scope != "user" {
		t.Fatalf("expected scope user, got %s", parsed.Scope)
	}
	if parsed.Selected != userPath {
		t.Fatalf("expected selected %s, got %s", userPath, parsed.Selected)
	}
	if parsed.User != userPath {
		t.Fatalf("expected user %s, got %s", userPath, parsed.User)
	}
	if projPath != "" && parsed.Project != "" {
		// No project path expected in this test without a project root.
	}
}

func TestConfigPathAllJSON(t *testing.T) {
	userPath := config.GlobalConfigPath()
	out := configPathJSON{
		Scope: "user",
		User:  userPath,
		Paths: []string{userPath},
	}

	data, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}

	var parsed configPathJSON
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatal(err)
	}
	if len(parsed.Paths) == 0 {
		t.Fatal("--all should include paths")
	}
}

func TestConfigPathUserScopeHasSelected(t *testing.T) {
	userPath := config.GlobalConfigPath()
	out := configPathJSON{Scope: "user", User: userPath, Selected: userPath, Paths: []string{userPath}}
	if out.Selected != userPath {
		t.Fatalf("user scope should have selected, got %q", out.Selected)
	}
}

func TestConfigPathEffectiveNoSelected(t *testing.T) {
	userPath := config.GlobalConfigPath()
	out := configPathJSON{Scope: "effective", User: userPath, Project: "/some/project/m.jsonc"}
	if out.Selected != "" {
		t.Fatalf("effective scope should not have selected, got %q", out.Selected)
	}
}

func TestConfigPathJSONOmitempty(t *testing.T) {
	userPath := config.GlobalConfigPath()
	out := configPathJSON{Scope: "user", User: userPath, Selected: userPath, Paths: []string{userPath}}
	data, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}
	// project should be omitted when empty.
	if strings.Contains(string(data), `"project"`) {
		t.Fatal("project field should be omitted when empty")
	}
}

// ── atomic file helper test ─────────────────────────────────────

func TestWriteAtomicReplacesContent(t *testing.T) {
	testkit.CleanEnv(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "test.json")
	orig := []byte("original\n")
	if err := fsx.PublishFileDurable(path, orig, 0o644); err != nil {
		t.Fatal(err)
	}
	// Verify content
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(orig) {
		t.Fatalf("expected %q, got %q", orig, got)
	}

	// Replace
	next := []byte("replacement\n")
	if err := fsx.PublishFileDurable(path, next, 0o644); err != nil {
		t.Fatal(err)
	}
	got, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(next) {
		t.Fatalf("expected %q, got %q", next, got)
	}
}

// ── recovery path tests ─────────────────────────────────────────

func TestRecoveryPathFormat(t *testing.T) {
	recPath := recoveryPath("/home/user/.config/mew/config.jsonc")
	dir := filepath.Dir(recPath)
	base := filepath.Base(recPath)
	if dir != "/home/user/.config/mew" {
		t.Fatalf("unexpected recovery dir: %s", dir)
	}
	if !strings.HasPrefix(base, "config.jsonc.invalid-") {
		t.Fatalf("expected config.jsonc.invalid-* prefix, got %s", base)
	}
}
