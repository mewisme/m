// Package transaction stages, journals, commits, and rolls back install mutations.
package transaction

import "context"

// Transaction is the sole commit path for install-family filesystem mutations.
type Transaction interface {
	Begin(ctx context.Context) error
	Stage(ctx context.Context) error
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}
