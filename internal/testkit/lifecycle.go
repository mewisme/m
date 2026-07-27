package testkit

import "testing"

// EnableLifecycle turns on MEW_EXPERIMENTAL_LIFECYCLE for lifecycle integration tests.
func EnableLifecycle(t testing.TB) {
	t.Helper()
	t.Setenv("MEW_EXPERIMENTAL_LIFECYCLE", "1")
}
