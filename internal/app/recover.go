package app

import (
	"context"

	"github.com/mewisme/m/internal/transaction"
)

// RecoverResult summarizes recovery actions.
type RecoverResult struct {
	Action string `json:"action"`
	TxnID  string `json:"txnId,omitempty"`
}

// Recover rolls back or discards an incomplete transaction (idempotent).
func Recover(ctx context.Context, ac *Context) (RecoverResult, error) {
	var out RecoverResult
	proj, err := OpenProject(ctx, ac)
	if err != nil {
		return out, err
	}
	txn, err := transaction.LoadIncomplete(proj.Root)
	if err != nil {
		return out, err
	}
	if txn == nil {
		out.Action = "none"
		return out, nil
	}
	out.TxnID = txn.ID
	doc := txn.Document()
	if doc == nil {
		out.Action = "none"
		return out, nil
	}
	switch doc.State {
	case transaction.StateStaging:
		if err := txn.Discard(); err != nil {
			return out, err
		}
		out.Action = "discarded"
	case transaction.StateValidated, transaction.StateCommitting:
		if err := txn.Rollback(ctx); err != nil {
			return out, err
		}
		_ = txn.Finish(false)
		out.Action = "rolled_back"
	default:
		out.Action = "none"
	}
	return out, nil
}
