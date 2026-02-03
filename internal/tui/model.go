package tui

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ivoronin/wch/internal/diff"
	"github.com/ivoronin/wch/internal/runner"
	"github.com/ivoronin/wch/internal/session"
	"github.com/ivoronin/wch/internal/tui/scrollview"
)

// Config holds TUI configuration.
type Config struct {
	Command        string
	Interval       time.Duration
	DiffEnabled    bool
	ShowStatus     bool
	NotifyOnChange bool
}

// Model is the Bubble Tea model.
type Model struct {
	// Core
	session *session.Session
	runner  *runner.Runner
	differ  *diff.Highlighter

	// View state
	historyIndex int                   // selected history item (-1 if empty)
	viewport     scrollview.Scrollview // content display
	mode         Mode                  // current interaction mode

	// Options
	diff      bool // show diff highlighting
	statusBar bool // show status bar
	notify    bool // send notifications on change
	paused    bool

	// Runtime
	executing bool
	width     int
	height    int
	ready     bool
}

// defaultMaxHistory retains last 1000 command executions (~8 hours at 30s interval)
const defaultMaxHistory = 1000

// New creates a new TUI model.
func New(cfg Config) Model {
	sess := session.NewSession(cfg.Command, cfg.Interval)
	sess.MaxHistory = defaultMaxHistory

	return Model{
		session:    sess,
		runner:     runner.New(cfg.Command),
		differ:     diff.New(),
		viewport:     scrollview.NewScrollview(0, 0),
		historyIndex: -1,
		mode:       viewMode,
		diff:       cfg.DiffEnabled,
		statusBar:  cfg.ShowStatus,
		notify:     cfg.NotifyOnChange,
	}
}

// Init starts the TUI.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		func() tea.Msg { return tickMsg{} },
		tea.SetWindowTitle("wch: "+m.session.Command),
	)
}

// Update handles Bubble Tea messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)
	case tea.WindowSizeMsg:
		return m.handleResize(msg)
	case tickMsg:
		return m.handleTick()
	case execResultMsg:
		return m.handleExecResult(msg)
	}
	return m, nil
}

// handleTick processes the periodic execution tick.
func (m Model) handleTick() (Model, tea.Cmd) {
	if m.paused {
		return m, m.scheduleNextTick()
	}
	m.executing = true
	return m, m.executeCmd()
}

// handleKey dispatches key events: global keys first, then mode-specific.
func (m Model) handleKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	if action := handleGlobalKey(msg); action != nil {
		return m.applyGlobalAction(action)
	}
	if action := m.mode.HandleKey(msg); action != nil {
		return m.applyModeAction(action)
	}
	return m, nil
}

// applyGlobalAction applies a global action message to the model.
func (m Model) applyGlobalAction(action tea.Msg) (Model, tea.Cmd) {
	switch action.(type) {
	case tea.QuitMsg:
		return m, tea.Quit
	case escapeMsg:
		m.mode = viewMode
		if n := len(m.session.History); n > 0 {
			m.historyIndex = n - 1
		}
		return m.withUpdatedContent(), nil
	case toggleDiffMsg:
		m.diff = !m.diff
		return m.withUpdatedContent(), nil
	case toggleBarMsg:
		m.statusBar = !m.statusBar
		return m.withResizedScrollview(), nil
	case togglePauseMsg:
		m.paused = !m.paused
	}
	return m, nil
}

// applyModeAction applies a mode action message to the model.
func (m Model) applyModeAction(action tea.Msg) (Model, tea.Cmd) {
	switch action := action.(type) {
	case scrollUpMsg:
		m.viewport.ScrollUp(action.lines)
	case scrollDownMsg:
		m.viewport.ScrollDown(action.lines)
	case scrollLeftMsg:
		m.viewport.ScrollLeft()
	case scrollRightMsg:
		m.viewport.ScrollRight()
	case pageUpMsg:
		m.viewport.PageUp()
	case pageDownMsg:
		m.viewport.PageDown()
	case scrollLeftPageMsg:
		m.viewport.ScrollLeftPage()
	case scrollRightPageMsg:
		m.viewport.ScrollRightPage()
	case gotoLeftEdgeMsg:
		m.viewport.GotoLeftEdge()
	case gotoRightEdgeMsg:
		m.viewport.GotoRightEdge()
	case switchModeMsg:
		// Guard: no picker mode without history
		if action.mode == pickerMode && len(m.session.History) == 0 {
			return m, nil
		}
		m.mode = action.mode
	case moveCursorMsg:
		return m.withCursor(m.historyIndex + action.delta), nil
	case gotoFirstCursorMsg:
		return m.withCursor(0), nil
	case gotoLastCursorMsg:
		return m.withCursor(len(m.session.History) - 1), nil
	}
	return m, nil
}

// handleResize responds to terminal size changes.
func (m Model) handleResize(msg tea.WindowSizeMsg) (Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height
	m.ready = true
	return m.withResizedScrollview(), nil
}

// handleExecResult processes command execution results.
func (m Model) handleExecResult(msg execResultMsg) (Model, tea.Cmd) {
	m.executing = false

	// Track if cursor was at the last item before adding new one
	wasAtLast := m.historyIndex == len(m.session.History)-1

	// Record execution
	added := m.session.RecordIfChanged(msg.exec)
	if added && wasAtLast {
		m.historyIndex = len(m.session.History) - 1
	}

	// Refresh content
	m = m.withUpdatedContent()

	// Schedule next tick and notify if changed
	var cmds []tea.Cmd
	if m.notify && added && len(m.session.History) > 1 {
		cmds = append(cmds, sendNotification())
	}
	cmds = append(cmds, m.scheduleNextTick())
	return m, tea.Batch(cmds...)
}

// executeCmd runs the command and returns the result as a message.
func (m Model) executeCmd() tea.Cmd {
	return func() tea.Msg {
		return execResultMsg{exec: m.runner.Execute(context.Background())}
	}
}

// scheduleNextTick schedules the next execution tick.
func (m Model) scheduleNextTick() tea.Cmd {
	return tea.Tick(m.session.Interval, func(time.Time) tea.Msg {
		return tickMsg{}
	})
}

// withUpdatedContent returns a model with refreshed scrollview content.
func (m Model) withUpdatedContent() Model {
	if m.historyIndex < 0 {
		return m
	}

	history := m.session.History
	exec := history[m.historyIndex]
	output := exec.Output()

	// Append error info
	if exec.Error != nil && exec.ExitCode != 0 {
		exitCodeMessage := errorStyle.Render(fmt.Sprintf("Exit code: %d", exec.ExitCode))
		if output == "" {
			output = exitCodeMessage
		} else {
			output += "\n" + exitCodeMessage
		}
	}

	// Apply diff highlighting
	if m.diff && m.historyIndex > 0 {
		prev := history[m.historyIndex-1].Output()
		output = m.differ.Highlight(prev, output)
	}

	m.viewport.SetContent(output)
	return m
}

// withResizedScrollview returns a model with updated viewport dimensions.
func (m Model) withResizedScrollview() Model {
	w, h := m.width, m.height
	if m.statusBar {
		h--
	}
	m.viewport.SetSize(w, h)
	return m
}

// withCursor returns a model with the cursor moved and content updated.
func (m Model) withCursor(newCursor int) Model {
	lastIdx := len(m.session.History) - 1
	m.historyIndex = max(0, min(lastIdx, newCursor))
	return m.withUpdatedContent()
}

// isFollowing returns true if viewing the latest history item.
func (m Model) isFollowing() bool {
	n := len(m.session.History)
	return n == 0 || m.historyIndex == n-1
}

// sendNotification sends an OSC9 notification using tea.Printf for TUI safety.
func sendNotification() tea.Cmd {
	return tea.Printf("\033]9;wch: output changed\a")
}
