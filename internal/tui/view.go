package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// View renders the UI
func (m Model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	var b strings.Builder

	// Main viewport content
	b.WriteString(m.viewport.View())

	// Status bar
	if m.showStatus {
		b.WriteString("\n")
		b.WriteString(m.renderStatusBar())
	}

	return b.String()
}

func (m Model) renderStatusBar() string {
	// Left side: indicator + command
	indicator := "·"
	if m.paused {
		indicator = "⏸"
	} else if m.isExecuting {
		indicator = "*"
	}
	leftPart := indicator + " " + m.session.Command

	// Right side: timestamp
	var rightPart string
	if exec := m.session.LastExecution(); exec != nil {
		rightPart = exec.Timestamp.Format("15:04:05")
	}

	// Calculate padding using display width (for Unicode support)
	leftWidth := lipgloss.Width(leftPart)
	rightWidth := lipgloss.Width(rightPart)
	paddingLen := m.width - leftWidth - rightWidth

	if paddingLen < 1 {
		// Truncate command if needed
		available := m.width - rightWidth - 6 // "* " + "..."
		if available > 0 && len(m.session.Command) > available {
			leftPart = indicator + " " + m.session.Command[:available] + "..."
			leftWidth = lipgloss.Width(leftPart)
			paddingLen = m.width - leftWidth - rightWidth
		}
	}

	if paddingLen < 0 {
		paddingLen = 0
	}

	padding := strings.Repeat(" ", paddingLen)
	fullBar := leftPart + padding + rightPart

	return statusBarStyle.Render(fullBar)
}
