package presentation

import "strings"

// TerminalIntent describes how a command should treat the terminal around children.
type TerminalIntent string

const (
	TerminalAuto        TerminalIntent = "auto"
	TerminalInteractive TerminalIntent = "interactive"
	TerminalStream      TerminalIntent = "stream"
)

// ResolveTerminalIntent picks intent from an explicit override or auto.
func ResolveTerminalIntent(explicit string) TerminalIntent {
	switch TerminalIntent(strings.ToLower(strings.TrimSpace(explicit))) {
	case TerminalInteractive:
		return TerminalInteractive
	case TerminalStream:
		return TerminalStream
	default:
		return TerminalAuto
	}
}

// ShouldSuspendRichUI reports whether live presentation must yield before a child.
// Auto is conservative: suspend when stdin, stdout, and stderr are all TTYs.
func ShouldSuspendRichUI(intent TerminalIntent, caps Capabilities) bool {
	switch intent {
	case TerminalInteractive:
		return true
	case TerminalStream:
		return false
	default: // auto
		return caps.StdinTTY && caps.StdoutTTY && caps.StderrTTY
	}
}

// WantLiveProgressForCommand reports whether this command may start live progress.
// Stream intent disables live progress for the command.
func WantLiveProgressForCommand(intent TerminalIntent) bool {
	return intent != TerminalStream
}
