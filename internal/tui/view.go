package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
)

// Constants are defined in styles.go

// View renders the UI.
func (m Model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	// Main content
	content := m.viewport.View()

	// Bottom bar: picker bar replaces status bar when in picker mode
	var bottomBar string
	if m.mode == pickerMode {
		bottomBar = m.renderPickerBar()
	} else if m.statusBar {
		bottomBar = m.renderStatusBar()
	}

	if bottomBar != "" {
		content += "\n" + bottomBar
	}

	return content
}

// renderStatusBar renders the bottom status bar.
// Layout: [command]  [timestamp][indicator]  [help]
// Timestamp is centered; indicator follows it; command/help fill sides.
func (m Model) renderStatusBar() string {
	timestamp := m.renderTimestamp()
	indicator := m.renderIndicator()

	// Side widths: center timestamp (not the whole center block)
	contentWidth := m.width - statusBarStyle.GetHorizontalFrameSize()
	leftWidth := (contentWidth - lipgloss.Width(timestamp)) / 2
	rightWidth := contentWidth - leftWidth - lipgloss.Width(timestamp) - lipgloss.Width(indicator)

	left := renderLeft(m.session.Command, leftWidth)
	right := renderRight(renderHelp(m.mode), rightWidth)
	content := lipgloss.JoinHorizontal(lipgloss.Top, left, timestamp, indicator, right)

	return statusBarStyle.Width(m.width).Render(content)
}

func (m Model) renderTimestamp() string {
	if m.historyIndex >= 0 {
		return m.session.History[m.historyIndex].Timestamp.Format(timestampFmt)
	}
	return ""
}

func (m Model) renderIndicator() string {
	var indicator string
	switch {
	case !m.isFollowing():
		indicator = "⎌"
	case m.paused:
		indicator = "⏸"
	case m.executing:
		indicator = "*"
	default:
		indicator = "·"
	}
	return indicatorStyle.Render(indicator)
}

// helpProvider is a minimal interface for help rendering.
type helpProvider interface {
	ShortHelp() []key.Binding
}

// renderHelp renders styled help bindings.
func renderHelp(hp helpProvider) string {
	bindings := hp.ShortHelp()
	parts := make([]string, 0, len(bindings))
	for _, b := range bindings {
		if !b.Enabled() {
			continue
		}
		h := b.Help()
		parts = append(parts, h.Key+" "+h.Desc)
	}
	return helpStyle.Render(strings.Join(parts, " • "))
}
