package tui

import (
	"context"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/ivoronin/wch/internal/diff"
	"github.com/ivoronin/wch/internal/runner"
	"github.com/ivoronin/wch/internal/session"
)

// Config holds TUI configuration
type Config struct {
	Command        string
	Interval       time.Duration
	DiffEnabled    bool
	ShowStatus     bool
	NotifyOnChange bool
}

// Model is the bubbletea model
type Model struct {
	// Core state
	session *session.Session
	runner  *runner.Runner
	differ  *diff.Highlighter

	// UI components
	viewport viewport.Model
	keys     keyMap

	// Cached view state
	prevOutput string
	styledView string

	// UI options
	diffEnabled    bool
	showStatus     bool
	notifyOnChange bool
	paused         bool

	// Runtime state
	isExecuting bool
	width       int
	height      int
	ready       bool
	xOffset     int
}

// Messages
type execResultMsg struct {
	exec session.Execution
}

type tickMsg struct{}

// New creates a new model
func New(cfg Config) Model {
	sess := session.NewSession(cfg.Command, cfg.Interval)
	sess.MaxHistory = 100

	return Model{
		session:        sess,
		runner:         runner.New(cfg.Command),
		differ:         diff.New(),
		keys:           newKeyMap(),
		diffEnabled:    cfg.DiffEnabled,
		showStatus:     cfg.ShowStatus,
		notifyOnChange: cfg.NotifyOnChange,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.executeCmd(),
		tea.SetWindowTitle("wch: "+m.session.Command),
	)
}

// executeCmd returns a command that executes the watched command
func (m Model) executeCmd() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		result := m.runner.Execute(ctx)
		return execResultMsg{exec: result}
	}
}

// scheduleNextTick returns a command that waits for the interval then sends a tick
func (m Model) scheduleNextTick() tea.Cmd {
	return tea.Tick(m.session.Interval, func(time.Time) tea.Msg {
		return tickMsg{}
	})
}
