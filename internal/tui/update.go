package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, m.keys.ToggleDiff):
			m.diffEnabled = !m.diffEnabled
			m.updateViewContent()
			return m, nil

		case key.Matches(msg, m.keys.ToggleBar):
			m.showStatus = !m.showStatus
			m.updateViewportSize()
			return m, nil

		case key.Matches(msg, m.keys.Pause):
			m.paused = !m.paused
			return m, nil

		case key.Matches(msg, m.keys.Up):
			m.viewport.ScrollUp(1)
			return m, nil

		case key.Matches(msg, m.keys.Down):
			m.viewport.ScrollDown(1)
			return m, nil

		case key.Matches(msg, m.keys.Left):
			m.scrollLeft()
			return m, nil

		case key.Matches(msg, m.keys.Right):
			m.scrollRight()
			return m, nil

		case key.Matches(msg, m.keys.PageUp):
			m.viewport.PageUp()
			return m, nil

		case key.Matches(msg, m.keys.PageDown):
			m.viewport.PageDown()
			return m, nil

		case key.Matches(msg, m.keys.Home):
			m.viewport.GotoTop()
			return m, nil

		case key.Matches(msg, m.keys.End):
			m.viewport.GotoBottom()
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		if !m.ready {
			m.viewport = viewport.New(msg.Width, m.viewportHeight())
			m.ready = true
		} else {
			m.updateViewportSize()
			// Re-render content and clamp scroll positions after resize
			if m.styledView != "" {
				m.refreshViewport()
			}
		}
		return m, tea.ClearScreen

	case execResultMsg:
		m.isExecuting = false
		m.session.AddExecution(msg.exec)
		m.updateViewContent()

		// Send notification if content changed
		if m.notifyOnChange && m.prevOutput != "" {
			newOutput := msg.exec.Output()
			if newOutput != m.prevOutput {
				cmds = append(cmds, m.sendNotification())
			}
		}
		m.prevOutput = msg.exec.Output()

		cmds = append(cmds, m.scheduleNextTick())
		return m, tea.Batch(cmds...)

	case tickMsg:
		if !m.paused {
			m.isExecuting = true
			return m, m.executeCmd()
		}
		return m, m.scheduleNextTick()

	}

	return m, nil
}

func (m *Model) viewportHeight() int {
	if m.showStatus {
		return m.height - 1
	}
	return m.height
}

func (m *Model) updateViewportSize() {
	m.viewport.Width = m.width
	m.viewport.Height = m.viewportHeight()
}

func (m *Model) refreshViewport() {
	m.setViewportContent(m.styledView)
}

func (m *Model) updateViewContent() {
	exec := m.session.LastExecution()
	if exec == nil {
		return
	}

	output := exec.Output()

	if exec.Error != nil && exec.ExitCode != 0 {
		output = fmt.Sprintf("%s\n\n%s", output, errorStyle.Render(fmt.Sprintf("Exit code: %d", exec.ExitCode)))
	}

	if m.diffEnabled && m.prevOutput != "" {
		highlighted, _ := m.differ.Highlight(m.prevOutput, output)
		m.styledView = highlighted
	} else {
		m.styledView = output
	}

	m.setViewportContent(m.styledView)
}

func (m *Model) setViewportContent(content string) {
	yOffset := m.viewport.YOffset

	m.viewport.SetContent(content)

	// Clamp vertical scroll
	maxY := max(0, m.viewport.TotalLineCount()-m.viewport.Height)
	m.viewport.YOffset = min(yOffset, maxY)

	// Clamp and restore horizontal scroll
	maxX := max(0, m.maxLineWidth()-m.viewport.Width)
	m.xOffset = min(m.xOffset, maxX)
	m.viewport.ScrollLeft(m.viewport.Width + m.maxLineWidth())
	if m.xOffset > 0 {
		m.viewport.ScrollRight(m.xOffset)
	}
}

func (m *Model) maxLineWidth() int {
	maxWidth := 0
	for _, line := range strings.Split(m.styledView, "\n") {
		if len(line) > maxWidth {
			maxWidth = len(line)
		}
	}
	return maxWidth
}

func (m *Model) scrollLeft() {
	if m.xOffset > 0 {
		m.xOffset--
		m.viewport.ScrollLeft(1)
	}
}

func (m *Model) scrollRight() {
	maxX := max(0, m.maxLineWidth()-m.viewport.Width)
	if m.xOffset < maxX {
		m.xOffset++
		m.viewport.ScrollRight(1)
	}
}

func (m Model) sendNotification() tea.Cmd {
	return func() tea.Msg {
		// OSC 9 notification
		fmt.Print("\033]9;wch: output changed\a")
		return nil
	}
}
