package tui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/ivoronin/wch/internal/tui/helprender"
	"github.com/ivoronin/wch/internal/tui/notify"
)

// View renders the UI. Layer order: viewport → bar → help overlay (when toggled) → notify
// bubbles. Notify draws last so a transient bubble still surfaces over the help panel.
func (m Model) View() tea.View {
	var content string
	if !m.ready {
		content = "Initializing..."
	} else {
		content = m.frames.View()
		if m.barShown() {
			if bar := m.state.RenderBar(m); bar != "" {
				content += "\n" + bar
			}
		}
		if m.prefs.HelpVisible {
			content = helprender.Overlay(content, renderHelpPanel(), m.width, m.height)
		}
		if m.notify.Active() {
			// Adaptive insets: snug to the bottom-right corner of the *available* content
			// area. We add an inset only when there's something to avoid overlaying — the
			// vertical scrollbar on the right, the status bar and/or horizontal scrollbar
			// at the bottom. No fixed gaps.
			var insets notify.Insets
			if m.frames.NeedsVerticalScrollbar() {
				insets.Right = 1
			}
			if m.barShown() {
				insets.Bottom++
			}
			if m.frames.NeedsHorizontalScrollbar() {
				insets.Bottom++
			}
			content = m.notify.Overlay(content, m.width, m.height, insets)
		}
	}

	v := tea.NewView(content)
	v.AltScreen = true
	if m.isLive() {
		v.WindowTitle = "wch: " + m.session.Command
	} else {
		v.WindowTitle = "wch (replay): " + m.session.Command
	}
	return v
}
