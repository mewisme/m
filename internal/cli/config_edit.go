package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/config"
	"github.com/mewisme/mew/internal/fsx"
)

// resolveEditorCommand resolves the editor command in VISUAL → EDITOR → platform
// fallback order. Returns the executable name and any arguments.
func resolveEditorCommand() (name string, args []string, err error) {
	var raw string
	if raw = os.Getenv("VISUAL"); raw != "" {
	} else if raw = os.Getenv("EDITOR"); raw != "" {
	} else {
		raw = platformEditorFallback()
	}
	if strings.TrimSpace(raw) == "" {
		return "", nil, apperr.New(apperr.Config, "config.edit", "editor",
			"no editor configured; set $VISUAL or $EDITOR")
	}
	name, args, err = splitEditorCommand(raw)
	if err != nil {
		return "", nil, apperr.Wrap(apperr.Config, "config.edit", raw, err)
	}
	if name == "" {
		return "", nil, apperr.New(apperr.Config, "config.edit", raw,
			"editor command is empty")
	}
	return name, args, nil
}

// platformEditorFallback returns the platform-specific default editor.
func platformEditorFallback() string {
	if runtime.GOOS == "windows" {
		return "notepad"
	}
	return "nano"
}

// splitEditorCommand splits an editor value like "code --wait" into executable
// and arguments. Handles single-quoted and double-quoted strings.
func splitEditorCommand(raw string) (name string, args []string, err error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil, nil
	}
	parts, err := shellSplit(raw)
	if err != nil {
		return "", nil, err
	}
	if len(parts) == 0 {
		return "", nil, nil
	}
	return parts[0], parts[1:], nil
}

// shellSplit splits a command line into tokens, handling single and double quotes.
// Does not handle backslash escapes within single quotes (literal) but does handle
// them within double quotes.
func shellSplit(s string) ([]string, error) {
	var out []string
	var cur strings.Builder
	inSingle := false
	inDouble := false
	escaped := false

	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case escaped:
			cur.WriteByte(c)
			escaped = false
		case inSingle:
			if c == '\'' {
				inSingle = false
			} else {
				cur.WriteByte(c)
			}
		case inDouble:
			if c == '\\' {
				escaped = true
			} else if c == '"' {
				inDouble = false
			} else {
				cur.WriteByte(c)
			}
		case c == '\'':
			inSingle = true
		case c == '"':
			inDouble = true
		case c == ' ' || c == '\t':
			if cur.Len() > 0 {
				out = append(out, cur.String())
				cur.Reset()
			}
		default:
			cur.WriteByte(c)
		}
	}

	if inSingle {
		return nil, fmt.Errorf("unterminated single quote in %q", s)
	}
	if inDouble {
		return nil, fmt.Errorf("unterminated double quote in %q", s)
	}
	if escaped {
		cur.WriteByte('\\')
	}
	if cur.Len() > 0 {
		out = append(out, cur.String())
	}
	return out, nil
}

// editConfigFile opens path in the user's editor, validates the result, and
// handles recovery when the edited content is invalid.
func editConfigFile(ctx context.Context, cmd *cobra.Command, g *globalFlags, targetPath string, scope config.Scope) error {
	// Resolve the editor command.
	editorName, editorArgs, err := resolveEditorCommand()
	if err != nil {
		return err
	}

	// Read original content for recovery.
	orig, origErr := os.ReadFile(targetPath)

	// Create file with {} if missing.
	if origErr != nil && errors.Is(origErr, os.ErrNotExist) {
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return apperr.Wrap(apperr.IO, "config.edit", targetPath, err)
		}
		if err := os.WriteFile(targetPath, []byte("{}\n"), 0o644); err != nil {
			return apperr.Wrap(apperr.IO, "config.edit", targetPath, err)
		}
		orig = []byte("{}\n")
	} else if origErr != nil {
		return apperr.Wrap(apperr.IO, "config.edit", targetPath, err)
	}

	// Run the editor.
	args := append(editorArgs, targetPath)
	ecmd := exec.CommandContext(ctx, editorName, args...)
	ecmd.Stdin = os.Stdin
	ecmd.Stdout = os.Stdout
	ecmd.Stderr = os.Stderr
	editErr := ecmd.Run()

	// Read the result.
	edited, readErr := os.ReadFile(targetPath)
	if readErr != nil {
		// Cannot read the edited file — restore original if possible.
		if origErr == nil {
			if restoreErr := fsx.PublishFileDurable(targetPath, orig, 0o644); restoreErr != nil {
				return apperr.Wrap(apperr.IO, "config.edit", targetPath,
					fmt.Errorf("cannot read edited file (%w) and restore failed (%w)", readErr, restoreErr))
			}
			return apperr.Wrap(apperr.IO, "config.edit", targetPath,
				fmt.Errorf("cannot read edited file, original restored: %w", readErr))
		}
		return apperr.Wrap(apperr.IO, "config.edit", targetPath, readErr)
	}

	// If the editor exited non-zero but the file changed, validate it anyway.
	if editErr != nil {
		// Check if the file actually changed.
		changed := !bytesEqual(orig, edited)
		if !changed {
			return apperr.Wrap(apperr.IO, "config.edit", editorName, editErr)
		}
		// File changed despite non-zero exit. Validate and handle.
		return handleEditedContent(targetPath, orig, edited, editorName, scope, editErr)
	}

	// Editor exited successfully. Check if unchanged.
	if bytesEqual(orig, edited) {
		return nil // no changes, no rewrite needed
	}

	return handleEditedContent(targetPath, orig, edited, editorName, scope, nil)
}

// handleEditedContent validates the edited bytes and handles recovery.
func handleEditedContent(targetPath string, orig, edited []byte, editorName string, scope config.Scope, editorErr error) error {
	// Validate the edited content using the shared validator.
	res := config.ValidateDocument(edited, targetPath, config.ValidateOptions{Scope: scope})

	if res.Valid() {
		// Content is valid. If editor exited non-zero, still accept valid content.
		if editorErr != nil {
			return apperr.Wrap(apperr.IO, "config.edit", editorName,
				fmt.Errorf("editor exited with error but file is valid: %w", editorErr))
		}
		// Content is valid and editor succeeded.
		return nil
	}

	// Invalid content: preserve at recovery path, restore original.
	recPath := recoveryPath(targetPath)

	// Validate that the edited content produced an error we can report.
	valErr := res.Err()
	if valErr == nil {
		valErr = apperr.New(apperr.Config, "config.edit", targetPath, "document is invalid")
	}

	// Try to write the invalid content to recovery path.
	if err := fsx.PublishNewFileExclusive(recPath, edited, 0o644); err != nil {
		// Recovery file creation failed — do not destroy edited content.
		// Return config error with both validation failure and recovery failure.
		return apperr.Wrap(apperr.Config, "config.edit", targetPath,
			fmt.Errorf("invalid config (recovery write to %s failed: %w); edited content remains at %s; validation error: %w",
				recPath, err, targetPath, valErr))
	}

	// Restore original atomically.
	if restoreErr := fsx.PublishFileDurable(targetPath, orig, 0o644); restoreErr != nil {
		msg := fmt.Sprintf("invalid config preserved at %s; original restore failed: %v; validation error: %v; manual recovery required",
			recPath, restoreErr, valErr)
		if editorErr != nil {
			msg += fmt.Sprintf("; editor also failed: %v", editorErr)
		}
		return apperr.New(apperr.Config, "config.edit", targetPath, msg)
	}

	msg := fmt.Sprintf("invalid config: %v; original restored, invalid content saved to %s", valErr, recPath)
	if editorErr != nil {
		msg += fmt.Sprintf("; editor also failed: %v", editorErr)
	}
	return apperr.New(apperr.Config, "config.edit", targetPath, msg)
}

// recoveryPath builds a unique recovery path for the given config file.
func recoveryPath(targetPath string) string {
	dir := filepath.Dir(targetPath)
	base := filepath.Base(targetPath)
	ts := time.Now().UTC().Format("20060102-150405")
	return filepath.Join(dir, base+".invalid-"+ts)
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
