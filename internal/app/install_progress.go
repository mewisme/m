package app

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/mewisme/mew/internal/diagnostics"
)

// Install phase names carried in OperationStarted.Kind.
const (
	phaseResolve   = "resolve"
	phaseFetch     = "fetch"
	phaseLink      = "link"
	phaseLifecycle = "lifecycle"
	phaseValidate  = "validate"
	phaseCommit    = "commit"
	phaseRollback  = "rollback"
	phaseCleanup   = "cleanup"
)

const (
	statusOK        = "ok"
	statusFailed    = "failed"
	statusCancelled = "cancelled"
	statusSkipped   = "skipped"
)

var installOpSeq atomic.Uint64

// installProgress emits typed Operation* events for one install phase.
// Domain-owned; presentation consumes via diagnostics.Reporter hooks.
type installProgress struct {
	ac     *Context
	opID   string
	phase  string
	id     string
	label  string
	unit   string
	total  *int64
	start  time.Time
	active bool
	done   bool
}

func newInstallOpID() string {
	n := installOpSeq.Add(1)
	return fmt.Sprintf("%d", n)
}

func beginInstallPhase(ac *Context, opID, phase string) *installProgress {
	return beginInstallPhaseLabeled(ac, opID, phase, phase, nil, "")
}

func beginInstallPhaseLabeled(ac *Context, opID, phase, label string, total *int64, unit string) *installProgress {
	if opID == "" {
		opID = newInstallOpID()
	}
	p := &installProgress{
		ac:    ac,
		opID:  opID,
		phase: phase,
		id:    "install/" + opID + "/" + phase,
		label: label,
		unit:  unit,
		total: total,
	}
	p.startPhase()
	return p
}

func (p *installProgress) startPhase() {
	if p == nil || p.done || p.active {
		return
	}
	p.start = time.Now()
	p.active = true
	if p.ac == nil || p.ac.Reporter == nil {
		return
	}
	ev := diagnostics.OperationStartedEvent{
		V:     1,
		Type:  "operation-started",
		ID:    p.id,
		Kind:  p.phase,
		Label: p.label,
		Total: p.total,
		Unit:  p.unit,
	}
	p.ac.Reporter.OperationStarted(ev)
}

// Progress emits OperationProgress when totals are authoritative.
func (p *installProgress) Progress(completed int64, detail string) {
	if p == nil || !p.active || p.done {
		return
	}
	if p.ac == nil || p.ac.Reporter == nil {
		return
	}
	p.ac.Reporter.OperationProgress(diagnostics.OperationProgressEvent{
		V:         1,
		Type:      "operation-progress",
		ID:        p.id,
		Completed: completed,
		Total:     p.total,
		Detail:    detail,
	})
}

// Complete finishes the phase with an explicit status. Duplicate completes are ignored.
func (p *installProgress) Complete(status string, metrics ...diagnostics.Metric) {
	if p == nil || p.done {
		return
	}
	if !p.active {
		return
	}
	p.done = true
	p.active = false
	if status == "" {
		status = statusOK
	}
	if p.ac == nil || p.ac.Reporter == nil {
		return
	}
	dur := time.Since(p.start).Milliseconds()
	p.ac.Reporter.OperationCompleted(diagnostics.OperationCompletedEvent{
		V:          1,
		Type:       "operation-completed",
		ID:         p.id,
		Status:     status,
		DurationMs: dur,
		Metrics:    metrics,
	})
	p.ac.Reporter.Debug("install phase",
		diagnostics.Attr{Key: "phase", Value: p.phase},
		diagnostics.Attr{Key: "elapsed", Value: (time.Duration(dur) * time.Millisecond).String()},
		diagnostics.Attr{Key: "status", Value: status},
	)
}

// CompleteErr maps err to failed/cancelled and completes the phase.
func (p *installProgress) CompleteErr(err error, metrics ...diagnostics.Metric) {
	p.Complete(statusFromErr(err), metrics...)
}

func statusFromErr(err error) string {
	if err == nil {
		return statusOK
	}
	if errors.Is(err, context.Canceled) {
		return statusCancelled
	}
	return statusFailed
}

// emitInstallNotice emits a typed Notice (lifecycle blocked/security).
func emitInstallNotice(ac *Context, severity, code, message, hint string) {
	if ac == nil || ac.Reporter == nil {
		return
	}
	ac.Reporter.Notice(diagnostics.NoticeEvent{
		V:        1,
		Type:     "notice",
		Severity: severity,
		Code:     code,
		Message:  message,
		Hint:     hint,
	})
}
