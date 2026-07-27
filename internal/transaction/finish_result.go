package transaction

// FinishResult reports post-mutation cleanup outcomes.
type FinishResult struct {
	Committed       bool
	LockReleased    bool
	CurrentCleared  bool
	CleanupWarnings []error
}

func (fr FinishResult) HasCriticalCleanupFailure() bool {
	return !fr.LockReleased || !fr.CurrentCleared
}
