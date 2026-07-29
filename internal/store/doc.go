// Package store is the content-addressed global package store.
//
// Blobs are keyed by digest (algo/hex) and verified on read, write, and
// existence checks via PutVerified, GetVerified, OpenVerified, and
// ExistsVerified. Corrupt on-disk blobs are quarantined rather than served.
package store
