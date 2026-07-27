package transaction

// CleanupSeverity classifies post-mutation cleanup outcomes.
type CleanupSeverity uint8

const (
	CleanupWarning CleanupSeverity = iota
	CleanupCritical
)

// CleanupCodeSeverity returns the severity for a cleanup warning code.
// Unknown codes default to non-critical (warning).
func CleanupCodeSeverity(code string) CleanupSeverity {
	switch code {
	case CleanupCodeTxnCurrentCleanup, CleanupCodeTxnLockRelease:
		return CleanupCritical
	case "txn_dir_remove", "finish_hook":
		return CleanupWarning
	case "store_import_lock_release", "store_index_lock_release":
		return CleanupWarning
	default:
		return CleanupWarning
	}
}
