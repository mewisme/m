package testkit

import "testing"

// EnableDirectScripts turns on MEW_EXPERIMENTAL_DIRECT_SCRIPTS for dispatch tests.
func EnableDirectScripts(t testing.TB) {
	t.Helper()
	t.Setenv("MEW_EXPERIMENTAL_DIRECT_SCRIPTS", "1")
}
