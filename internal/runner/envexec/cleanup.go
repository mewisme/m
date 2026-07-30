package envexec

import "context"

// CleanupFunc releases ephemeral resources for a prepared environment.
//
// Contract:
//   - Idempotent: safe to call more than once.
//   - Invoked exactly once by Orchestrator after execution completes.
//   - Uses a bounded cleanup context independent of a canceled execution context.
//   - Primary prepare/execute error wins over cleanup error.
//   - Cleanup error is attached as a diagnostic when a primary error exists.
//   - If execution succeeds but mandatory cleanup fails, return ERR_M_IO.
//   - Best-effort access-metadata cleanup failure is warning-only.
//   - Shared immutable cache roots are not deleted after execution.
type CleanupFunc func(ctx context.Context) error
