package cli

import (
	"os"
)

// RuntimeEnabled reports whether the experimental runtime file-run feature is on.
func RuntimeEnabled() bool {
	return os.Getenv("MEW_EXPERIMENTAL_RUNTIME") == "1"
}
