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
	txns, err := transaction.ScanIncompleteTxns(proj.Root)
	if err != nil {
		return out, err
	}
	auth, err := transaction.ResolveAuthoritativeIncomplete(txns)
	if err != nil {
		return out, err
	}
	if auth == nil {
		out.Action = "none"
		return out, nil
	}
	out.TxnID = auth.ID
	switch auth.State {
	case transaction.StateStaging, transaction.StateValidated:
		out.Action = "discarded"
	case transaction.StateCommitting:
		out.Action = "rolled_back"
	default:
		out.Action = "none"
	}
	if err := transaction.RecoverScanned(ctx, proj.Root, transaction.RecoverScannedOpts{}); err != nil {
		return out, err
	}
	again, err := transaction.ScanIncompleteTxns(proj.Root)
	if err != nil {
		return out, err
	}
	if len(again) > 0 {
		return out, nil
	}
	return out, nil
}
