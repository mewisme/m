package presentation

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"

	"github.com/mewisme/mew/internal/diagnostics"
)

// LiveInstallRenderer is an inline Bubble Tea progress sink for stderr.
// It does not use the alternate screen and does not own stdin.
// The tea program starts lazily on the first progress event so commands that
// never emit progress (version, help, config) do not touch the terminal.
type LiveInstallRenderer struct {
	mu       sync.Mutex
	w        io.Writer
	settings EffectiveSettings
	p        *tea.Program
	done     chan struct{}
	runErr   error
	closed   bool
	started  bool
	suspend  bool
}

type livePhase struct {
	Kind      string
	Label     string
	Status    string // pending|active|ok|failed|cancelled|skipped
	Completed int64
	Total     *int64
	Detail    string
}

type liveModel struct {
	spinner  spinner.Model
	settings EffectiveSettings
	phases   []livePhase
	byID     map[string]int
	notices  []string
	suspend  bool
	quitting bool
	width    int
}

type (
	liveStartedMsg   diagnostics.OperationStartedEvent
	liveProgressMsg  diagnostics.OperationProgressEvent
	liveCompletedMsg diagnostics.OperationCompletedEvent
	liveNoticeMsg    diagnostics.NoticeEvent
	liveSuspendMsg   struct{}
	liveResumeMsg    struct{}
)

// NewLiveInstallRenderer prepares an inline tea progress sink writing to w (stderr).
// The program is not started until the first progress event.
func NewLiveInstallRenderer(w io.Writer, settings EffectiveSettings) (*LiveInstallRenderer, error) {
	if w == nil {
		w = io.Discard
	}
	return &LiveInstallRenderer{
		w:        w,
		settings: settings,
	}, nil
}

func (r *LiveInstallRenderer) OperationStarted(ev diagnostics.OperationStartedEvent) {
	r.send(liveStartedMsg(ev), true)
}

func (r *LiveInstallRenderer) OperationProgress(ev diagnostics.OperationProgressEvent) {
	r.send(liveProgressMsg(ev), true)
}

func (r *LiveInstallRenderer) OperationCompleted(ev diagnostics.OperationCompletedEvent) {
	r.send(liveCompletedMsg(ev), true)
}

func (r *LiveInstallRenderer) Notice(ev diagnostics.NoticeEvent) {
	r.send(liveNoticeMsg(ev), true)
}

func (r *LiveInstallRenderer) Suspend() {
	r.send(liveSuspendMsg{}, false)
}

func (r *LiveInstallRenderer) Resume() {
	r.send(liveResumeMsg{}, false)
}

func (r *LiveInstallRenderer) Close() error {
	r.mu.Lock()
	if r.closed {
		err := r.runErr
		r.mu.Unlock()
		return err
	}
	r.closed = true
	p := r.p
	done := r.done
	r.mu.Unlock()
	if p == nil || done == nil {
		return nil
	}
	p.Quit()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		p.Kill()
		<-done
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.runErr
}

func (r *LiveInstallRenderer) send(msg tea.Msg, start bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return
	}
	switch msg.(type) {
	case liveSuspendMsg:
		r.suspend = true
	case liveResumeMsg:
		r.suspend = false
	}
	if !r.started {
		if !start {
			return
		}
		r.startLocked()
	}
	if r.p == nil {
		return
	}
	r.p.Send(msg)
}

func (r *LiveInstallRenderer) startLocked() {
	sp := spinner.New()
	if r.settings.UseUnicode {
		sp.Spinner = spinner.MiniDot
	} else {
		sp.Spinner = spinner.Line
	}
	m := liveModel{
		spinner:  sp,
		settings: r.settings,
		byID:     map[string]int{},
		width:    r.settings.Width,
		suspend:  r.suspend,
	}
	p := tea.NewProgram(m,
		tea.WithOutput(r.w),
		tea.WithInput(nil),
		tea.WithoutSignals(),
		tea.WithEnvironment(liveInstallEnviron(os.Environ())),
	)
	done := make(chan struct{})
	r.p = p
	r.done = done
	r.started = true
	go func() {
		_, err := p.Run()
		r.mu.Lock()
		r.runErr = err
		r.mu.Unlock()
		close(done)
	}()
}

// liveInstallEnviron builds the Bubble Tea environment for a no-stdin live sink.
//
// ponytail: Bubble Tea v2 emits DEC mode 2026/2027 capability queries on start.
// WithInput(nil) cannot consume the replies, so they leak into the shell as
// literal input ([?2027;3$y]). shouldQuerySynchronizedOutput is false for
// Apple Terminal over SSH; spoof that pair so queries are skipped.
// Upgrade: upstream option to disable probes when input is nil.
func liveInstallEnviron(base []string) []string {
	out := make([]string, 0, len(base)+2)
	for _, e := range base {
		switch {
		case strings.HasPrefix(e, "WT_SESSION="),
			strings.HasPrefix(e, "SSH_TTY="),
			strings.HasPrefix(e, "TERM_PROGRAM="):
			continue
		default:
			out = append(out, e)
		}
	}
	out = append(out, "SSH_TTY=1", "TERM_PROGRAM=Apple_Terminal")
	return out
}

func (m liveModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m liveModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.QuitMsg:
		m.quitting = true
		return m, nil
	case liveSuspendMsg:
		m.suspend = true
		return m, nil
	case liveResumeMsg:
		m.suspend = false
		return m, m.spinner.Tick
	case liveStartedMsg:
		if m.suspend {
			return m, nil
		}
		kind := strings.TrimSpace(msg.Kind)
		if kind == "" {
			kind = strings.TrimSpace(msg.Label)
		}
		idx, ok := m.byID[msg.ID]
		if !ok {
			idx = len(m.phases)
			m.byID[msg.ID] = idx
			m.phases = append(m.phases, livePhase{})
		}
		m.phases[idx] = livePhase{
			Kind:   kind,
			Label:  msg.Label,
			Status: "active",
			Total:  msg.Total,
		}
		return m, nil
	case liveProgressMsg:
		if m.suspend {
			return m, nil
		}
		if idx, ok := m.byID[msg.ID]; ok {
			m.phases[idx].Completed = msg.Completed
			m.phases[idx].Total = msg.Total
			m.phases[idx].Detail = msg.Detail
			m.phases[idx].Status = "active"
		}
		return m, nil
	case liveCompletedMsg:
		if idx, ok := m.byID[msg.ID]; ok {
			st := strings.TrimSpace(msg.Status)
			if st == "" {
				st = "ok"
			}
			m.phases[idx].Status = st
		}
		return m, nil
	case liveNoticeMsg:
		line := strings.TrimSpace(msg.Message)
		if line != "" {
			m.notices = append(m.notices, line)
		}
		return m, nil
	case spinner.TickMsg:
		if m.suspend || m.quitting {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m liveModel) View() tea.View {
	if m.quitting && len(m.phases) == 0 && len(m.notices) == 0 {
		return tea.NewView("")
	}
	sym := m.settings.Symbols
	var b strings.Builder
	for _, ph := range m.phases {
		label := ph.Label
		if label == "" {
			label = ph.Kind
		}
		var mark string
		switch ph.Status {
		case "active":
			if m.suspend {
				mark = " "
			} else {
				mark = m.spinner.View()
			}
		case "ok":
			mark = sym.Success
			if mark == "" {
				mark = "OK"
			}
		case "failed":
			mark = sym.Error
			if mark == "" {
				mark = "X"
			}
		case "cancelled":
			mark = sym.Warning
			if mark == "" {
				mark = "!"
			}
		case "skipped":
			mark = sym.Info
			if mark == "" {
				mark = "-"
			}
		default:
			mark = " "
		}
		b.WriteString(mark)
		b.WriteByte(' ')
		b.WriteString(label)
		if ph.Total != nil && *ph.Total > 0 {
			fmt.Fprintf(&b, "  %d/%d", ph.Completed, *ph.Total)
		} else if ph.Status == "active" || ph.Status == "pending" {
			b.WriteString("  pending")
		}
		b.WriteByte('\n')
	}
	for _, n := range m.notices {
		b.WriteString(sym.Warning)
		if sym.Warning != "" {
			b.WriteByte(' ')
		}
		b.WriteString(n)
		b.WriteByte('\n')
	}
	v := tea.NewView(strings.TrimRight(b.String(), "\n"))
	v.AltScreen = false
	return v
}
