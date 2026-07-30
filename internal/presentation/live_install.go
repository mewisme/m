package presentation

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"

	"github.com/mewisme/mew/internal/diagnostics"
)

// LiveInstallRenderer is an inline Bubble Tea progress sink for stderr.
// It does not use the alternate screen and does not own stdin.
type LiveInstallRenderer struct {
	mu       sync.Mutex
	p        *tea.Program
	done     chan struct{}
	runErr   error
	closed   bool
	settings EffectiveSettings
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

// NewLiveInstallRenderer starts an inline tea program writing to w (stderr).
func NewLiveInstallRenderer(w io.Writer, settings EffectiveSettings) (*LiveInstallRenderer, error) {
	if w == nil {
		w = io.Discard
	}
	sp := spinner.New()
	if settings.UseUnicode {
		sp.Spinner = spinner.MiniDot
	} else {
		sp.Spinner = spinner.Line
	}
	m := liveModel{
		spinner:  sp,
		settings: settings,
		byID:     map[string]int{},
		width:    settings.Width,
	}
	p := tea.NewProgram(m,
		tea.WithOutput(w),
		tea.WithInput(nil),
		tea.WithoutSignals(),
	)
	r := &LiveInstallRenderer{
		p:        p,
		done:     make(chan struct{}),
		settings: settings,
	}
	go func() {
		_, err := p.Run()
		r.mu.Lock()
		r.runErr = err
		r.mu.Unlock()
		close(r.done)
	}()
	return r, nil
}

func (r *LiveInstallRenderer) OperationStarted(ev diagnostics.OperationStartedEvent) {
	r.send(liveStartedMsg(ev))
}

func (r *LiveInstallRenderer) OperationProgress(ev diagnostics.OperationProgressEvent) {
	r.send(liveProgressMsg(ev))
}

func (r *LiveInstallRenderer) OperationCompleted(ev diagnostics.OperationCompletedEvent) {
	r.send(liveCompletedMsg(ev))
}

func (r *LiveInstallRenderer) Notice(ev diagnostics.NoticeEvent) {
	r.send(liveNoticeMsg(ev))
}

func (r *LiveInstallRenderer) Suspend() {
	r.send(liveSuspendMsg{})
}

func (r *LiveInstallRenderer) Resume() {
	r.send(liveResumeMsg{})
}

func (r *LiveInstallRenderer) Close() error {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return r.runErr
	}
	r.closed = true
	p := r.p
	r.mu.Unlock()
	if p != nil {
		p.Quit()
	}
	select {
	case <-r.done:
	case <-time.After(2 * time.Second):
		if p != nil {
			p.Kill()
		}
		<-r.done
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.runErr
}

func (r *LiveInstallRenderer) send(msg tea.Msg) {
	r.mu.Lock()
	closed := r.closed
	p := r.p
	r.mu.Unlock()
	if closed || p == nil {
		return
	}
	p.Send(msg)
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
			b.WriteString(fmt.Sprintf("  %d/%d", ph.Completed, *ph.Total))
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
