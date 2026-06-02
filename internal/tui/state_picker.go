package tui

import (
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/ivoronin/wch/internal/session"
)

// timestampLen is the display width of a formatted timestamp.
var timestampLen = lipgloss.Width(timestampFmt)

// pickerSpacerLeft / pickerSpacerRight are hoisted style values used per item in the picker
// timeline. Lipgloss styles are immutable so the per-render NewStyle().Padding...() calls
// were pure waste — these constants do the same work once at startup.
var (
	pickerSpacerLeft  = lipgloss.NewStyle().PaddingRight(itemSpacing)
	pickerSpacerRight = lipgloss.NewStyle().PaddingLeft(itemSpacing)
)

// Body for pickerState is identical to viewState: the picker timeline lives in the bar,
// the viewport shows the frame at the cursor position.
func (pickerState) Body(m Model) (string, bool) {
	i, ok := m.cursor.At()
	return m.frames.Frame(i, m.prefs.Diff), ok
}

// Timestamp returns the timestamp of the frame at the cursor position -- the picker's
// cursor and view's cursor are the same field on Model.
func (pickerState) Timestamp(m Model) (time.Time, bool) {
	i, ok := m.cursor.At()
	if !ok {
		return time.Time{}, false
	}
	return m.session.History[i].Timestamp, true
}

// ShowsBar returns true because the picker timeline IS the bar — without it
// the user has no visible cursor or surrounding context for the history
// position they are navigating.
func (pickerState) ShowsBar() bool { return true }

// IsFrozen returns false: picker shows the same body as view, and that body
// must keep tracking live executions while the user browses history.
func (pickerState) IsFrozen() bool { return false }

// FollowsTail mirrors viewState: the underlying historyIndex advances to the
// new tail iff the user was at the tail. The picker's own cursor sits on
// historyIndex by definition.
func (pickerState) FollowsTail(wasAtTail bool) bool { return wasAtTail }

// RenderBar renders the timeline-style bar replacing the status bar: a left
// and right column of timestamps surrounding the centered selected timestamp.
func (pickerState) RenderBar(m Model) string {
	return renderPickerTimeline(m.session.History, m.cursor.Index(), m.width)
}

// Handle processes a key for pickerState. Common bindings (diff/pause/record/
// search) are tried first via handleCommonKey; picker-specific bindings
// (Confirm, cursor movement) come after. ↑/↓ are unhandled here and fall
// through to global scroll.
func (s pickerState) Handle(m Model, msg tea.KeyPressMsg) (Model, state, tea.Cmd, bool) {
	if newM, st, cmd, handled := m.handleCommonKey(s, msg); handled {
		return newM, st, cmd, true
	}
	switch {
	case key.Matches(msg, pickerKeys.Confirm), key.Matches(msg, commonKeys.Escape):
		return m, viewState{}, nil, true
	case key.Matches(msg, navKeys.Left):
		return m.withCursor(m.cursor.Index() - 1), s, nil, true
	case key.Matches(msg, navKeys.Right):
		return m.withCursor(m.cursor.Index() + 1), s, nil, true
	case key.Matches(msg, navKeys.Home), key.Matches(msg, navKeys.ScrollLeft):
		return m.withCursor(0), s, nil, true
	case key.Matches(msg, navKeys.End), key.Matches(msg, navKeys.ScrollRight):
		return m.withCursor(len(m.session.History) - 1), s, nil, true
	}
	return m, s, nil, false
}

// renderPickerTimeline is the pure-data picker bar renderer: given the history slice, the
// selected index, and the available width, it builds the horizontal timeline strip.
func renderPickerTimeline(history []session.Execution, selected, width int) string {
	if len(history) == 0 {
		return statusBarStyle.Width(width).Render("")
	}

	itemWidth := timestampLen + itemSpacing

	timestamp := history[selected].Timestamp.Format(timestampFmt)
	layout := calcThreeColumnLayout(width, timestampLen)

	leftItems, rightItems := pickerItems(history, selected, layout.leftWidth-arrowWidth, layout.rightWidth-arrowWidth, itemWidth)

	left := pickerSide(layout.leftWidth, leftItems, selected > len(leftItems), true)
	right := pickerSide(layout.rightWidth, rightItems, selected+len(rightItems)+1 < len(history), false)

	center := pickerSelectedStyle.Render(timestamp)
	content := lipgloss.JoinHorizontal(lipgloss.Top, left, center, right)
	return statusBarStyle.Width(width).Render(content)
}

// pickerItems returns the timestamps that fit in the left/right sections around the selection.
func pickerItems(history []session.Execution, selected, leftSpace, rightSpace, itemWidth int) (left, right []string) {
	for i := selected - 1; i >= 0 && leftSpace >= itemWidth; i-- {
		left = append(left, history[i].Timestamp.Format(timestampFmt))
		leftSpace -= itemWidth
	}
	for i := selected + 1; i < len(history) && rightSpace >= itemWidth; i++ {
		right = append(right, history[i].Timestamp.Format(timestampFmt))
		rightSpace -= itemWidth
	}
	return left, right
}

// pickerSide renders one side of the picker timeline (left or right of the selected
// timestamp). The two sides are mirror images: items run inward toward the centered
// selection, an arrow appears at the outer edge when more entries exist beyond the visible
// window. left=true renders the left side (items reversed, spacer pads on the right of each
// item, arrow at the left, items right-aligned); left=false renders the right side.
func pickerSide(width int, items []string, more, left bool) string {
	arrow := lipgloss.NewStyle().Width(arrowWidth).Render("")
	if more {
		if left {
			arrow = "◀ "
		} else {
			arrow = " ▶"
		}
	}

	spacer := pickerSpacerRight
	if left {
		spacer = pickerSpacerLeft
	}
	spacedItems := make([]string, len(items))
	for i, item := range items {
		idx := i
		if left {
			idx = len(items) - 1 - i
		}
		spacedItems[idx] = spacer.Render(item)
	}
	itemsStr := lipgloss.JoinHorizontal(lipgloss.Top, spacedItems...)

	if left {
		content := renderRight(itemsStr, width-arrowWidth)
		return pickerItemStyle.Render(lipgloss.JoinHorizontal(lipgloss.Top, arrow, content))
	}
	content := renderLeft(itemsStr, width-arrowWidth)
	return pickerItemStyle.Render(lipgloss.JoinHorizontal(lipgloss.Top, content, arrow))
}
