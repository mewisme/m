package transaction_test

import (
	"context"
	"testing"

	"github.com/mewisme/m/internal/transaction"
)

type fakeTxn struct{}

func (fakeTxn) Begin(context.Context) error    { return nil }
func (fakeTxn) Stage(context.Context) error    { return nil }
func (fakeTxn) Commit(context.Context) error   { return nil }
func (fakeTxn) Rollback(context.Context) error { return nil }

var _ transaction.Transaction = fakeTxn{}

func TestFakeTransactionSatisfiesInterface(t *testing.T) {
	var tx transaction.Transaction = fakeTxn{}
	ctx := context.Background()
	if err := tx.Begin(ctx); err != nil {
		t.Fatal(err)
	}
	if err := tx.Stage(ctx); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
}
