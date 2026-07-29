package transaction

import (
	"fmt"
	"os"
	"strings"
	"sync"
)

var (
	testHookMu sync.Mutex
	testHook   func(phase string, opIndex int) error
)

// Test hook phases (SetTestHook / MEW_TXN_CRASH_AT):
// post_resolve, post_fetch, post_link, post_lockfile, post_validate,
// post_patch_copy, post_patch_preflight, post_patch_apply, post_patch_publish.

// SetTestHook registers a failure-injection hook for tests (phase, opIndex).
// Pass nil to clear.
func SetTestHook(fn func(phase string, opIndex int) error) {
	testHookMu.Lock()
	defer testHookMu.Unlock()
	testHook = fn
}

// InvokeTestHook runs the registered test hook (exported for app-layer injection points).
func InvokeTestHook(phase string, opIndex int) error {
	return invokeTestHook(phase, opIndex)
}

func invokeTestHook(phase string, opIndex int) error {
	if spec := strings.TrimSpace(os.Getenv("MEW_TXN_CRASH_AT")); spec != "" {
		key := fmt.Sprintf("%s:%d", phase, opIndex)
		if spec == phase || spec == key {
			os.Exit(2)
		}
	}
	testHookMu.Lock()
	fn := testHook
	testHookMu.Unlock()
	if fn == nil {
		return nil
	}
	return fn(phase, opIndex)
}
