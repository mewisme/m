package store

import "context"

// Key identifies an immutable store entry.
type Key string

// Store gets and puts content-addressed blobs.
type Store interface {
	Get(ctx context.Context, key Key) ([]byte, error)
	Put(ctx context.Context, key Key, content []byte) error
}
