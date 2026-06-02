// Package tui — handlers shared across multiple state types live here so they
// don't get filed under any single state's _.go file.
package tui

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

// handleCommonKey handles the diff/pause/record/search bindings shared by
// viewState and pickerState. Returns handled=false if msg matches none of
// them. Lives here (not in either state's file) because both states call it
// and neither owns the shape.
func (m Model) handleCommonKey(s state, msg tea.KeyPressMsg) (Model, state, tea.Cmd, bool) {
	switch {
	case key.Matches(msg, commonKeys.ToggleDiff):
		// Same frame → anchor would be identity. Skip the diff.Align dance and re-render
		// directly; scroll position stays put because viewport's offset is not touched.
		m.prefs.Diff = !m.prefs.Diff
		return m.repaint(), s, nil, true
	case key.Matches(msg, commonKeys.Pause):
		m.prefs.Paused = !m.prefs.Paused
		return m, s, nil, true
	case key.Matches(msg, commonKeys.Record):
		m2, st, cmd := m.toggleRecord(s)
		return m2, st, cmd, true
	case key.Matches(msg, commonKeys.Search):
		m2, st, cmd := m.openSearchInput(s)
		return m2, st, cmd, true
	}
	return m, s, nil, false
}
