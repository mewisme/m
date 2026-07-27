package transaction

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mewisme/m/internal/apperr"
)

// CurrentCleanupResult reports verified removal of transaction current metadata.
type CurrentCleanupResult struct {
	Verified bool
}

func clearCurrentVerified(projectRoot string) (CurrentCleanupResult, error) {
	var result CurrentCleanupResult
	var errs []error
	dir := TxnRoot(projectRoot)

	headPath := filepath.Join(dir, currentHeadName)
	if err := os.Remove(headPath); err != nil && !os.IsNotExist(err) {
		errs = append(errs, apperr.Wrap(apperr.IO, "transaction.current", headPath, err))
	}

	legacyPath := CurrentPath(projectRoot)
	if err := os.Remove(legacyPath); err != nil && !os.IsNotExist(err) {
		errs = append(errs, apperr.Wrap(apperr.IO, "transaction.current", legacyPath, err))
	}

	entries, readErr := os.ReadDir(dir)
	if readErr != nil {
		if !os.IsNotExist(readErr) {
			errs = append(errs, apperr.Wrap(apperr.IO, "transaction.current", dir, readErr))
		}
	} else {
		for _, ent := range entries {
			name := ent.Name()
			if !strings.HasPrefix(name, "current.") || name == currentHeadName {
				continue
			}
			if !validCurrentGenName(name) {
				errs = append(errs, apperr.New(apperr.Integrity, "transaction.current", name, "malformed current generation file"))
				continue
			}
			if err := os.Remove(filepath.Join(dir, name)); err != nil && !os.IsNotExist(err) {
				errs = append(errs, apperr.Wrap(apperr.IO, "transaction.current", name, err))
			}
		}
	}

	if len(errs) > 0 {
		return result, errors.Join(errs...)
	}

	id, readErr := readCurrentGeneration(projectRoot)
	if readErr != nil {
		return result, readErr
	}
	if id != "" {
		return result, apperr.New(apperr.Integrity, "transaction.current", projectRoot, "current pointer still present after cleanup")
	}
	result.Verified = true
	return result, nil
}

// ClearCurrentVerifiedForTest exposes clearCurrentVerified for external tests.
func ClearCurrentVerifiedForTest(projectRoot string) (CurrentCleanupResult, error) {
	return clearCurrentVerified(projectRoot)
}

func validCurrentGenName(name string) bool {
	if name == currentHeadName {
		return false
	}
	mid := strings.TrimPrefix(name, "current.")
	if mid == name {
		return false
	}
	_, err := strconv.ParseUint(mid, 10, 64)
	return err == nil
}
