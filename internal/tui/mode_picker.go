package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss" // used for Width() in layout calculation
)

// timestampLen is the display width of a formatted timestamp
var timestampLen = lipgloss.Width(timestampFmt)

// pickerMode is the mode for browsing execution history.
type pickerModeType struct{}

var pickerMode Mode = pickerModeType{}

// Key bindings are defined in keys.go

func (pickerModeType) ShortHelp() []key.Binding {
	return nil
}

func (pickerModeType) HandleKey(msg tea.KeyMsg) tea.Msg {
	switch {
	case key.Matches(msg, pickerKeys.Confirm):
		return switchModeMsg{mode: viewMode}
	case key.Matches(msg, navKeys.Left):
		return moveCursorMsg{delta: -1}
	case key.Matches(msg, navKeys.Right):
		return moveCursorMsg{delta: +1}
	case key.Matches(msg, viewKeys.Home), key.Matches(msg, viewKeys.ScrollLeft):
		return gotoFirstCursorMsg{}
	case key.Matches(msg, viewKeys.End), key.Matches(msg, viewKeys.ScrollRight):
		return gotoLastCursorMsg{}
	}
	return nil
}

// renderPickerBar renders the horizontal picker bar that replaces the status bar.
// Uses the same three-section layout as renderStatusBar for consistent positioning.
func (m Model) renderPickerBar() string {
	history := m.session.History
	if len(history) == 0 {
		return statusBarStyle.Width(m.width).Render("")
	}

	itemWidth := timestampLen + itemSpacing // "HH:MM:SS" + spacing

	// Same center calculation as renderStatusBar
	timestamp := history[m.historyIndex].Timestamp.Format(timestampFmt)
	layout := calcThreeColumnLayout(m.width, lipgloss.Width(timestamp))

	// Collect items that fit in each section
	leftItems, rightItems := m.collectPickerItems(layout.leftWidth-arrowWidth, layout.rightWidth-arrowWidth, itemWidth)

	// Build sections using lipgloss alignment
	left := m.renderPickerLeft(layout.leftWidth, leftItems)
	right := m.renderPickerRight(layout.rightWidth, rightItems)

	center := pickerSelectedStyle.Render(timestamp)
	content := lipgloss.JoinHorizontal(lipgloss.Top, left, center, right)
	return statusBarStyle.Width(m.width).Render(content)
}

// collectPickerItems returns timestamps that fit in left/right sections.
func (m Model) collectPickerItems(leftSpace, rightSpace, itemWidth int) ([]string, []string) {
	history := m.session.History

	var leftItems []string
	for i := m.historyIndex - 1; i >= 0 && leftSpace >= itemWidth; i-- {
		leftItems = append(leftItems, history[i].Timestamp.Format(timestampFmt))
		leftSpace -= itemWidth
	}

	var rightItems []string
	for i := m.historyIndex + 1; i < len(history) && rightSpace >= itemWidth; i++ {
		rightItems = append(rightItems, history[i].Timestamp.Format(timestampFmt))
		rightSpace -= itemWidth
	}

	return leftItems, rightItems
}

// renderPickerLeft renders: [arrow][items right-aligned toward center]
func (m Model) renderPickerLeft(width int, items []string) string {
	arrow := strings.Repeat(" ", arrowWidth)
	// Show left arrow if there are more items to the left than we can display
	if m.historyIndex > len(items) {
		arrow = "◀ "
	}

	// Build items (oldest to newest, with trailing spacing)
	spacer := lipgloss.NewStyle().PaddingRight(itemSpacing)
	spacedItems := make([]string, len(items))
	for i, item := range items {
		spacedItems[len(items)-1-i] = spacer.Render(item)
	}
	itemsStr := lipgloss.JoinHorizontal(lipgloss.Top, spacedItems...)

	content := renderRight(itemsStr, width-arrowWidth)
	return pickerItemStyle.Render(lipgloss.JoinHorizontal(lipgloss.Top, arrow, content))
}

// renderPickerRight renders: [items left-aligned from center][arrow]
func (m Model) renderPickerRight(width int, items []string) string {
	arrow := strings.Repeat(" ", arrowWidth)
	// Show right arrow if there are more items to the right than we can display
	if m.historyIndex+len(items)+1 < len(m.session.History) {
		arrow = " ▶"
	}

	// Build items (with leading spacing)
	spacer := lipgloss.NewStyle().PaddingLeft(itemSpacing)
	spacedItems := make([]string, len(items))
	for i, item := range items {
		spacedItems[i] = spacer.Render(item)
	}
	itemsStr := lipgloss.JoinHorizontal(lipgloss.Top, spacedItems...)

	content := renderLeft(itemsStr, width-arrowWidth)
	return pickerItemStyle.Render(lipgloss.JoinHorizontal(lipgloss.Top, content, arrow))
}
