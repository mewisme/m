package runner

import (
	"github.com/mewisme/mew/internal/diagnostics"
)

func emitWorkspaceTask(rep diagnostics.Reporter, pkg, script, status string, index, exit int) {
	if rep == nil {
		return
	}
	rep.WorkspaceTask(diagnostics.WorkspaceTaskEvent{
		V:       1,
		Type:    "workspace-task",
		Phase:   "workspace-run",
		Package: pkg,
		Script:  script,
		Status:  status,
		Index:   index,
		Exit:    exitPtr(exit, status),
	})
}

func emitWorkspaceSummary(rep diagnostics.Reporter, completed, failed, cancelled, skipped, notRun, effConc int) {
	if rep == nil {
		return
	}
	rep.WorkspaceSummary(diagnostics.WorkspaceSummaryEvent{
		V:                    1,
		Type:                 "workspace-summary",
		Phase:                "workspace-run",
		Completed:            completed,
		Failed:               failed,
		Cancelled:            cancelled,
		Skipped:              skipped,
		NotRun:               notRun,
		EffectiveConcurrency: effConc,
	})
}

func exitPtr(exit int, status string) *int {
	if exit != 0 || status == "done" || status == "fail" || status == "skip" {
		v := exit
		return &v
	}
	return nil
}

// BuildWorkspaceTaskEvent constructs a control-plane workspace-task event.
func BuildWorkspaceTaskEvent(task WorkspaceTask, script string, status TaskStatus, exitCode int) diagnostics.WorkspaceTaskEvent {
	return diagnostics.WorkspaceTaskEvent{
		V:       1,
		Type:    "workspace-task",
		Phase:   "workspace-run",
		Package: task.Name,
		Script:  script,
		Status:  string(status),
		Index:   task.Index,
		Exit:    exitPtr(exitCode, string(status)),
	}
}

// BuildChildOutputEvent constructs a structured child-output event.
func BuildChildOutputEvent(task WorkspaceTask, script, stream, message string, partial bool, seq *int) diagnostics.ChildOutputEvent {
	return diagnostics.ChildOutputEvent{
		V:       1,
		Type:    "child-output",
		Package: task.Name,
		Script:  script,
		Stream:  stream,
		Message: message,
		Partial: partial,
		Seq:     seq,
	}
}

// EmitWorkspaceTask emits a control-plane event when a reporter is configured.
func EmitWorkspaceTask(rep diagnostics.Reporter, task WorkspaceTask, script string, status TaskStatus, exitCode int) {
	if rep == nil {
		return
	}
	rep.WorkspaceTask(BuildWorkspaceTaskEvent(task, script, status, exitCode))
}

// EmitChildOutput emits structured child output (never raw bytes on structured stdout).
func EmitChildOutput(rep diagnostics.Reporter, mode WorkspaceOutputMode, task WorkspaceTask, script, stream, message string, partial bool, seq *int) {
	if rep == nil {
		return
	}
	rep.ChildOutput(BuildChildOutputEvent(task, script, stream, message, partial, seq), diagnostics.WorkspaceOutputMode(mode))
}
