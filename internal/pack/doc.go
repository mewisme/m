// Package pack implements sandboxed npm-compatible tarball packing.
//
// Paths are confined to the package root, symlinks and reparse points are
// rejected, and output is written atomically with size and count limits.
package pack
