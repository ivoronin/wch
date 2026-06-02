package tui

import (
	"fmt"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/ivoronin/wch/internal/tui/searchrender"
)

// searchPromptLabel is what appears in front of the editable search query in inputState
// when opened via the search flow.
const searchPromptLabel = "/"

// Body for searchState returns the captured body with the selected match highlighted.
// It deliberately does NOT scroll the viewport -- that would let a bar toggle ('t') or a
// terminal resize yank the user's scroll position back to the match even when they had
// scrolled away to read context. Scroll-to-match happens explicitly via the snap()
// target passed to repaintWith at search entry and on n/p navigation.
func (s searchState) Body(m Model) (string, bool) {
	if len(s.matches) == 0 {
		return "", false
	}
	sel := s.matches[s.selected]
	return searchrender.Render(s.body, sel.line, sel.col, sel.length), true
}

// Timestamp returns the captured frame's timestamp (set at search entry from the cursor's
// then-current frame). The clock stays pinned to the frame the user searched in even as
// new executions advance the underlying cursor in the background.
func (s searchState) Timestamp(m Model) (time.Time, bool) {
	if s.captured.IsZero() {
		return time.Time{}, false
	}
	return s.captured, true
}

// snap returns the viewport scroll-to-cell effect for the currently selected match,
// or nil when no matches. Used at search entry and on n/p navigation -- never from
// Body, so unrelated re-paints (bar toggle, resize) don't disturb the user's scroll
// position.
func (s searchState) snap() *snapTarget {
	if len(s.matches) == 0 {
		return nil
	}
	sel := s.matches[s.selected]
	return &snapTarget{line: sel.line, col: sel.col, length: sel.length}
}

// ShowsBar returns false: search hides under -t by design (the user opted out
// of the bar, and the highlighted match in the viewport remains visible).
// Model.barShown ORs this with m.prefs.StatusBar at the caller, so search's bar
// appears exactly when the user-configured preference is on. The return value
// is prev-agnostic: even in the search-from-picker flow (where prev is
// pickerState) the bar still hides — search's UX takes precedence over the
// picker's bar need.
func (searchState) ShowsBar() bool { return false }

// IsFrozen returns true: searchState's viewport holds a captured body that
// the user navigates with n/p. New history must not repaint — otherwise the
// highlight overlay disappears mid-navigation.
func (searchState) IsFrozen() bool { return true }

// FollowsTail returns the captured follow flag (set at search entry from the
// user's at-tail state). New history advances historyIndex in the background
// so an Esc from search lands at the new tail without a second keystroke.
func (s searchState) FollowsTail(_ bool) bool { return s.follow }

// RenderBar composes the three-column status bar with search-specific slot
// contents: the query and the match counter sit together at the bar's left
// edge with a single-space separator; when the query is too long to fit
// alongside the counter in the left zone, the query is ellipsized so the
// counter always stays visible.
func (s searchState) RenderBar(m Model) string {
	query := searchPromptLabel + s.query
	// Brackets give the counter just enough visual weight to read as a discrete element
	// without leaning on color or bold — those would compete with the timestamp+❄ group.
	counter := fmt.Sprintf("[%d of %d]", s.selected+1, len(s.matches))
	indicator := indicatorStyle.Render("❄")
	left := queryWithCounter(query, counter, barLeftZoneWidth(m.width))
	return m.renderBarLayout(left, indicator, renderHelp(searchHelpBindings()))
}

// Handle processes a key for searchState. n/p navigates matches with wrap;
// '/' opens a new search input (replacing this search on the stack); Esc
// pops back to prev. Unhandled keys fall through to globalKeys so navigation
// defaults scroll the frozen body without losing the highlight.
func (s searchState) Handle(m Model, msg tea.KeyPressMsg) (Model, state, tea.Cmd, bool) {
	switch {
	case key.Matches(msg, searchKeys.NavNext):
		m, s = s.advance(m, 1)
		return m, s, nil, true
	case key.Matches(msg, searchKeys.NavPrev):
		m, s = s.advance(m, -1)
		return m, s, nil, true
	case key.Matches(msg, commonKeys.Search):
		m2, st, cmd := m.openSearchInput(s)
		return m2, st, cmd, true
	case key.Matches(msg, commonKeys.Escape):
		return m, s.prev, nil, true
	}
	return m, s, nil, false
}

// advance moves the selection by delta (wrapping around), repaints the highlight,
// and snaps the viewport to the new match. Shared by NavNext and NavPrev.
func (s searchState) advance(m Model, delta int) (Model, searchState) {
	n := len(s.matches)
	s.selected = ((s.selected+delta)%n + n) % n
	return m.repaintWith(s, s.snap()), s
}

// searchHelpBindings is the bar trailer for searchState. n is advertised since it's the
// primary post-match action; the rest (p, / restart) lives in the help overlay.
func searchHelpBindings() []key.Binding {
	return minimalBarBindings(commonKeys.Escape, searchKeys.NavNext)
}

// queryWithCounter joins head and tail with a single space, ellipsizing head if the pair
// doesn't fit in zoneWidth. The tail is treated as a hard requirement; if the zone is too
// narrow to fit even tail+space+1 cell of head, the tail is returned alone.
func queryWithCounter(head, tail string, zoneWidth int) string {
	tailW := lipgloss.Width(tail)
	if zoneWidth <= tailW+1 {
		return tail
	}
	headBudget := zoneWidth - tailW - 1
	head = ansi.Truncate(head, headBudget, "…")
	return lipgloss.JoinHorizontal(lipgloss.Top, head, " ", tail)
}
