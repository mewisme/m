package transaction

import (
	"path/filepath"
	"strings"

	"github.com/mewisme/m/internal/apperr"
)

// GuardPath ensures rel is a safe path under projectRoot (no escapes via .. or symlinks).
func GuardPath(projectRoot, rel string) (string, error) {
	if projectRoot == "" {
		return "", apperr.New(apperr.Transaction, "transaction.path", rel, "empty project root")
	}
	rel = filepath.Clean(rel)
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", apperr.New(apperr.Transaction, "transaction.path", rel, "path escapes project root")
	}
	absRoot, err := filepath.Abs(projectRoot)
	if err != nil {
		return "", apperr.Wrap(apperr.Transaction, "transaction.path", projectRoot, err)
	}
	absRoot, err = filepath.EvalSymlinks(absRoot)
	if err != nil {
		return "", apperr.Wrap(apperr.Transaction, "transaction.path", projectRoot, err)
	}
	joined := filepath.Join(absRoot, rel)
	joined, err = filepath.Abs(joined)
	if err != nil {
		return "", apperr.Wrap(apperr.Transaction, "transaction.path", rel, err)
	}
	if joined != absRoot && !strings.HasPrefix(joined, absRoot+string(filepath.Separator)) {
		return "", apperr.New(apperr.Transaction, "transaction.path", rel, "path escapes project root")
	}
	return joined, nil
}
