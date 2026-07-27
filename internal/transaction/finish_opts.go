package transaction

// FinishOpts controls post-mutation cleanup performed by Runner finish paths.
type FinishOpts struct {
	ReleaseProjectLock bool
	ClearCurrent       bool
}

// DefaultFinishOpts is for runners under an outer MutationSession that owns the lock.
func DefaultFinishOpts() FinishOpts {
	return FinishOpts{ReleaseProjectLock: false, ClearCurrent: true}
}

// StandaloneFinishOpts releases the project lock and clears current metadata.
func StandaloneFinishOpts() FinishOpts {
	return FinishOpts{ReleaseProjectLock: true, ClearCurrent: true}
}

// RecoveryFinishOpts selects cleanup for RecoverScanned when the caller already holds the lock.
func RecoveryFinishOpts(skipTakeover bool) FinishOpts {
	opts := DefaultFinishOpts()
	if !skipTakeover {
		opts.ReleaseProjectLock = true
	}
	return opts
}
