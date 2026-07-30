package envexec

import "context"

// AcquireLeaseForTest exposes lease acquisition for conformance tests.
func (o *Orchestrator) AcquireLeaseForTest(ctx context.Context, env PreparedEnvironment) (func(), error) {
	return o.acquireLease(ctx, env)
}

// EmitPreparedForTest exposes prepared emission for conformance tests.
func (o *Orchestrator) EmitPreparedForTest(req ExecutionRequest, env PreparedEnvironment) error {
	return o.emitPrepared(req, env)
}

// RunCleanupForTest exposes cleanup for conformance tests.
func (o *Orchestrator) RunCleanupForTest(env PreparedEnvironment) {
	o.runCleanup(env)
}
