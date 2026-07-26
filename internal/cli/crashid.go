package cli

import (
	"fmt"
	"time"
)

func newCrashID() string {
	return fmt.Sprintf("crash-%d", time.Now().UnixNano())
}
