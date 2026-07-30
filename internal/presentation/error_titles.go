package presentation

import "github.com/mewisme/mew/internal/apperr"

// TitleForCode returns a stable human title for an error code.
func TitleForCode(code apperr.Code) string {
	if title, ok := errorTitles[code]; ok {
		return title
	}
	return "Operation failed"
}

var errorTitles = map[apperr.Code]string{
	apperr.OK:                  "Success",
	apperr.Usage:               "Invalid command usage",
	apperr.Cancelled:           "Operation cancelled",
	apperr.Internal:            "Mew encountered an internal error",
	apperr.InternalPanic:       "Mew encountered an internal error",
	apperr.IO:                  "Filesystem operation failed",
	apperr.Config:              "Configuration is invalid",
	apperr.Network:             "Network operation failed",
	apperr.Integrity:           "Integrity verification failed",
	apperr.Lockfile:            "Lockfile validation failed",
	apperr.LockUnsupported:     "Lockfile format is not supported",
	apperr.LockAmbiguous:       "Lockfile producer version is ambiguous",
	apperr.LockUnrepresentable: "Lockfile cannot be represented safely",
	apperr.Unimplemented:       "Command is not implemented",
	apperr.Unsupported:         "Operation is not supported",
	apperr.Manifest:            "Package manifest is invalid",
	apperr.NotFound:            "Required item was not found",
	apperr.Resolve:             "Dependency resolution failed",
	apperr.Install:             "Installation failed",
	apperr.Transaction:         "Project update failed",
	apperr.Store:               "Content store operation failed",
	apperr.Policy:              "Operation blocked by policy",
	apperr.PNPUnsupported:      "PnP install is not supported",
	apperr.Exec:                "Command execution failed",
	apperr.Timeout:             "Operation timed out",
}
