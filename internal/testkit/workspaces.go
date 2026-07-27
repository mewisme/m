package testkit

import "testing"

// EnableWorkspaces turns on MEW_EXPERIMENTAL_WORKSPACES for workspace integration tests.
func EnableWorkspaces(t testing.TB) {
	t.Helper()
	t.Setenv("MEW_EXPERIMENTAL_WORKSPACES", "1")
	t.Setenv("MEW_EXPERIMENTAL_ISOLATED_LINKER", "1")
}
