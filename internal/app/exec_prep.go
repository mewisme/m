package app

import (
	"time"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/diagnostics"
)

// emitProjectExecPrep prints a thin human prep banner for local run/exec without inventing EnvironmentPrepared.
func emitProjectExecPrep(ac *Context, command, packageName string) {
	if ac == nil || ac.Reporter == nil {
		return
	}
	title := "Running"
	if command != "" {
		title = "Running " + command
	}
	pkg := packageName
	if pkg == "" {
		pkg = command
	}
	ac.Reporter.Progress(diagnostics.Event{
		V:       1,
		Type:    "exec-prep",
		Phase:   title,
		Package: pkg,
		Message: "project",
	})
}

func emitExecCompletion(ac *Context, name string, dur time.Duration, exit int, runErr error) {
	if ac == nil || ac.Reporter == nil {
		return
	}
	status := "ok"
	if runErr != nil {
		if apperr.CodeOf(runErr) == apperr.Cancelled {
			status = "cancelled"
		} else {
			status = "fail"
		}
	} else if exit != 0 {
		status = "fail"
	}
	ac.Reporter.Progress(diagnostics.Event{
		V:       1,
		Type:    "exec-summary",
		Phase:   name,
		Status:  status,
		Exit:    exit,
		Package: name,
		Bytes:   dur.Milliseconds(),
	})
}
