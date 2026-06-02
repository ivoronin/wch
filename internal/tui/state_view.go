package tui

import (
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

// Body returns the rendered frame at the cursor position. Reports ok=false when there is
// no history yet so the caller leaves the viewport untouched. Frame returns "" for i<0,
// matching the no-cursor case.
func (viewState) Body(m Model) (string, bool) {
	i, ok := m.cursor.At()
	return m.frames.Frame(i, m.prefs.Diff), ok
}

// Timestamp returns the timestamp of the frame at the cursor position. Reports ok=false
// when there is no history yet.
func (viewState) Timestamp(m Model) (time.Time, bool) {
	i, ok := m.cursor.At()
	if !ok {
		return time.Time{}, false
	}
	return m.session.History[i].Timestamp, true
}

// ShowsBar returns false because viewState's bar visibility depends only on
// the user preference m.prefs.StatusBar — there is no view-driven need for the bar
// to stay on. Model.barShown ORs this with m.prefs.StatusBar at the caller.
func (viewState) ShowsBar() bool { return false }

// IsFrozen returns false: viewState always repaints on a new exec; the user
// is watching live output.
func (viewState) IsFrozen() bool { return false }

// FollowsTail returns the caller's wasAtTail: viewState advances historyIndex
// to the new tail iff the user was already at the tail before the new exec
// arrived.
func (viewState) FollowsTail(wasAtTail bool) bool { return wasAtTail }

// RenderBar composes the standard three-column status bar for viewState.
// renderBarLayout → renderLeft already truncates to its leftWidth slot, so no
// pre-truncate is needed here.
func (viewState) RenderBar(m Model) string {
	return m.renderBarLayout(m.session.Command, m.renderIndicator(), renderHelp(viewHelpBindings(m)))
}

// Handle processes a key for viewState. Common bindings (diff/pause/record/
// search) are tried first via handleCommonKey; viewState-specific bindings
// come after. Unmatched keys fall through to handleGlobalKey (q/t/navigation)
// via dispatchKey's `if !handled` path.
func (s viewState) Handle(m Model, msg tea.KeyPressMsg) (Model, state, tea.Cmd, bool) {
	if newM, st, cmd, handled := m.handleCommonKey(s, msg); handled {
		return newM, st, cmd, true
	}
	switch {
	case key.Matches(msg, viewKeys.Picker):
		if len(m.session.History) == 0 {
			return m, s, nil, true
		}
		return m, pickerState{}, nil, true
	case key.Matches(msg, commonKeys.Escape):
		n := len(m.session.History)
		if !m.cursor.Following(n) {
			return m.withCursor(n - 1), s, nil, true
		}
		return m, s, nil, true
	}
	return m, s, nil, false
}

// viewHelpBindings returns the bar trailer for viewState. At the live tail we advertise
// b to enter the picker; when viewing a past frame we swap that for Esc (back to tail).
func viewHelpBindings(m Model) []key.Binding {
	if m.isFollowing() {
		return minimalBarBindings(viewKeys.Picker)
	}
	return minimalBarBindings(commonKeys.Escape)
}
