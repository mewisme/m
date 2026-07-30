package fsx

import (
	"context"
	"os"
	"time"
)

const (
	renameRetryMaxAttempts = 12
	renameRetryBaseDelay   = 25 * time.Millisecond
	renameRetryMaxDelay    = 500 * time.Millisecond
)

// RenamePath moves src to dst, retrying transient Windows sharing violations.
func RenamePath(ctx context.Context, src, dst string) error {
	return renameWithRetry(ctx, src, dst)
}

func renameWithRetry(ctx context.Context, src, dst string) error {
	var last error
	delay := renameRetryBaseDelay
	for attempt := 1; attempt <= renameRetryMaxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		last = renamePath(ctx, src, dst)
		if last == nil {
			return nil
		}
		if !isTransientRenameErr(last) || attempt == renameRetryMaxAttempts {
			return NewRenameError("rename", src, dst, last)
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
		if delay < renameRetryMaxDelay {
			delay *= 2
			if delay > renameRetryMaxDelay {
				delay = renameRetryMaxDelay
			}
		}
	}
	if last == nil {
		return nil
	}
	return NewRenameError("rename", src, dst, last)
}

// RemoveAllRetry removes path, retrying transient Windows delete failures.
func RemoveAllRetry(ctx context.Context, path string) error {
	var last error
	delay := renameRetryBaseDelay
	for attempt := 1; attempt <= renameRetryMaxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		last = os.RemoveAll(path)
		if last == nil || os.IsNotExist(last) {
			return nil
		}
		if !isTransientRenameErr(last) || attempt == renameRetryMaxAttempts {
			return last
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
		if delay < renameRetryMaxDelay {
			delay *= 2
			if delay > renameRetryMaxDelay {
				delay = renameRetryMaxDelay
			}
		}
	}
	return last
}
