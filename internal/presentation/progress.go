package presentation

import "github.com/mewisme/mew/internal/diagnostics"

// ProgressSink observes install Operation* / Notice events for live or plain progress.
type ProgressSink interface {
	OperationStarted(diagnostics.OperationStartedEvent)
	OperationProgress(diagnostics.OperationProgressEvent)
	OperationCompleted(diagnostics.OperationCompletedEvent)
	Notice(diagnostics.NoticeEvent)
	Suspend()
	Resume()
	Close() error
}

// attachProgressHooks wires a ProgressSink into diagnostics Options callbacks.
func attachProgressHooks(opts *diagnostics.Options, sink ProgressSink) {
	if opts == nil || sink == nil {
		return
	}
	opts.OnOperationStarted = sink.OperationStarted
	opts.OnOperationProgress = sink.OperationProgress
	opts.OnOperationCompleted = sink.OperationCompleted
	opts.OnNotice = sink.Notice
}
