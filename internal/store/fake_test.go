package store_test

import (
	"context"
	"testing"

	"github.com/mewisme/mew/internal/store"
)

type fakeStore struct{}

func (fakeStore) Get(context.Context, store.Key) ([]byte, error) { return nil, nil }
func (fakeStore) Put(context.Context, store.Key, []byte) error   { return nil }

var _ store.Store = fakeStore{}

func TestFakeStoreSatisfiesInterface(t *testing.T) {
	var s store.Store = fakeStore{}
	if err := s.Put(context.Background(), "k", []byte("v")); err != nil {
		t.Fatal(err)
	}
}
