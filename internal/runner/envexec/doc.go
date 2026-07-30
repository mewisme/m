// Package envexec provides a shared execution-environment layer for m exec, mx,
// snapshot execution, and capsule execution.
//
// Dependency direction (enforced by import_test.go):
//
//	internal/cli → internal/app → internal/runner/envexec → internal/runner
//
// envexec must not import internal/app or internal/cli.
package envexec
