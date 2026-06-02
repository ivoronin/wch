package tui

import (
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"
)

// barShown reports whether the bottom bar takes up a screen row right now —
// either because the user enabled it (m.prefs.StatusBar) or because the current
// state needs unconditional bar feedback (signalled by state.ShowsBar). Each
// state decides for itself: inputState always wants the bar regardless of
// host; searchState's ShowsBar is false so search hides under -t by design,
// consistent with the user's "I asked for it off" intent.
func (m Model) barShown() bool {
	return m.prefs.StatusBar || m.state.ShowsBar()
}

// renderBarLayout composes the standard three-column bar from the given slot contents. Left
// is right-aligned within its zone (truncated/padded), the centered timestamp + activity
// indicator stay centered (with the optional REC dot charged to the left so the clock does
// not drift), and the help text fills the right zone.
func (m Model) renderBarLayout(leftContent, indicator, helpText string) string {
	layout := calcThreeColumnLayout(m.width, centerBlockWidth)
	left := barInnerStyle.Render(renderLeft(leftContent, layout.leftWidth))
	center := m.renderCenterBlock(indicator)
	right := barInnerStyle.Render(renderRight(helpText, layout.rightWidth))
	content := lipgloss.JoinHorizontal(lipgloss.Top, left, center, right)
	return statusBarStyle.Width(m.width).Render(content)
}

// centerBlockWidth is the fixed cell width of the centered clock-group:
//
//	[2 left-pad][rec slot 1][1 gap][timestamp][1 gap][indicator 1][2 right-pad]
//
// Each adjacent element on the bar has a 1-cell base gap; REC adds an extra cell on its
// left, indicator adds an extra cell on its right — so the clock group is bracketed by
// 2-cell gutters and stays visually distinct from the help and content zones. REC and
// indicator slots are always-on (blank-padded when REC is idle) so toggling recording
// mid-session never re-flows the left zone. Derived from timestampLen so a change to
// timestampFmt propagates without re-deriving constants.
var centerBlockWidth = 2 + 1 + 1 + timestampLen + 1 + 1 + 2

// renderCenterBlock composes the always-on clock-group at exactly centerBlockWidth visible
// cells: REC slot stays 1 cell whether recording or not. Gaps and pads are rendered inline
// per call so an adaptive-theme switch is picked up on the next frame.
func (m Model) renderCenterBlock(indicator string) string {
	oneSpace := barInnerStyle.Width(1).Render("")
	twoSpaces := barInnerStyle.Width(2).Render("")
	rec := oneSpace
	if m.flow.IsActive() {
		rec = recStyle.Render("●")
	}
	timestamp := barInnerStyle.Width(timestampLen).Render(m.renderTimestamp())
	return lipgloss.JoinHorizontal(lipgloss.Top,
		twoSpaces, rec, oneSpace, timestamp, oneSpace, indicator, twoSpaces,
	)
}

// renderTimestamp formats the timestamp the active state asks for in the bar center.
// view/picker return the frame at the cursor; search returns the captured snapshot;
// input delegates to its prev.
func (m Model) renderTimestamp() string {
	t, ok := m.state.Timestamp(m)
	if !ok {
		return ""
	}
	return t.Format(timestampFmt)
}

// renderIndicator renders the activity indicator shown in the bar center (right of the
// clock). searchState replaces it with ❄ via its own RenderBar.
func (m Model) renderIndicator() string {
	var indicator string
	switch {
	case !m.isLive():
		indicator = "▶"
	case !m.isFollowing():
		indicator = "⎌"
	case m.prefs.Paused:
		indicator = "⏸"
	case m.executing:
		indicator = "*"
	default:
		indicator = "·"
	}
	return indicatorStyle.Render(indicator)
}

// renderHelp builds the bar's right-side hint as "<bold key> desc" segments joined with
// " • " and wrapped in helpStyle. Keys are bolded via boldKeep so the outer fg/bg survive
// to the end of the line.
func renderHelp(bindings []key.Binding) string {
	parts := make([]string, len(bindings))
	for i, b := range bindings {
		h := b.Help()
		parts[i] = boldKeep(h.Key) + " " + h.Desc
	}
	return helpStyle.Render(strings.Join(parts, " • "))
}
