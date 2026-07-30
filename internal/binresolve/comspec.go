package binresolve

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
)

// ResolveComSpec returns a validated absolute ComSpec path. Never uses exec.LookPath or project PATH.
func ResolveComSpec(hostEnv []string) (string, error) {
	if runtime.GOOS != "windows" {
		return "", apperr.New(apperr.Exec, "binresolve.comspec", "", "ComSpec only applies on Windows")
	}
	if v, ok := lookupEnv(hostEnv, "ComSpec"); ok && strings.TrimSpace(v) != "" {
		return validateComSpec(v)
	}
	root := os.Getenv("SystemRoot")
	if root == "" {
		root = `C:\Windows`
	}
	return validateComSpec(filepath.Join(root, "System32", "cmd.exe"))
}

func validateComSpec(path string) (string, error) {
	if !filepath.IsAbs(path) {
		return "", apperr.New(apperr.Exec, "binresolve.comspec", path, "ComSpec must be absolute")
	}
	abs := filepath.Clean(path)
	st, err := os.Stat(abs)
	if err != nil {
		return "", apperr.Wrap(apperr.Exec, "binresolve.comspec", abs, err)
	}
	if st.IsDir() {
		return "", apperr.New(apperr.Exec, "binresolve.comspec", abs, "ComSpec is not a file")
	}
	base := strings.ToLower(filepath.Base(abs))
	if base != "cmd.exe" {
		return "", apperr.New(apperr.Exec, "binresolve.comspec", abs, "unsupported shell executable")
	}
	return abs, nil
}
