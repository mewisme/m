package app

import (
	"context"

	"github.com/mewisme/mew/internal/transaction"
)

// RecoverResult summarizes recovery actions.
type RecoverResult struct {
	Action string `json:"action"`
	TxnID  string `json:"txnId,omitempty"`
}

// Recover rolls back or discards an incomplete transaction and clears stale committed metadata (idempotent).
func Recover(ctx context.Context, ac *Context) (RecoverResult, error) {
	var out RecoverResult
	proj, err := OpenProject(ctx, ac)
	if err != nil {
		return out, err
	}
	txns, err := transaction.ScanIncompleteTxns(proj.Root)
	if err != nil {
		return out, err
	}
	auth, err := transaction.ResolveAuthoritativeIncomplete(txns)
	if err != nil {
		return out, err
	}
	if auth != nil {
		out.TxnID = auth.ID
		switch auth.State {
		case transaction.StateStaging, transaction.StateValidated:
			out.Action = "discarded"
		case transaction.StateCommitting:
			out.Action = "rolled_back"
		default:
			out.Action = "none"
		}
	}
	if err := transaction.RecoverScanned(ctx, proj.Root, transaction.RecoverScannedOpts{}); err != nil {
		return out, err
	}
	cleaned, err := transaction.RecoverCommittedCleanup(ctx, proj.Root)
	if err != nil {
		return out, err
	}
	if cleaned > 0 {
		if out.Action == "none" || out.Action == "" {
			out.Action = "committed_cleanup"
		}
	}
	again, err := transaction.ScanIncompleteTxns(proj.Root)
	if err != nil {
		return out, err
	}
	if len(again) > 0 {
		return out, nil
	}
	stale, err := transaction.ScanCommittedStale(proj.Root)
	if err != nil {
		return out, err
	}
	if len(stale) == 0 && out.Action == "" {
		out.Action = "none"
	}
	return out, nil
}
