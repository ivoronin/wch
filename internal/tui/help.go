package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

// helpBinding is a single entry in the help overlay's keybinding table. keys is the
// human-readable key (or composite like "Enter b" / "Shift+←→"); desc is the description.
type helpBinding struct {
	keys string
	desc string
}

// helpSection groups bindings under a heading.
type helpSection struct {
	title    string
	bindings []helpBinding
}

// helpSections returns the static keybinding reference shown in the help overlay. Not
// auto-derived from keys.go: composite key strings ("Enter b", "Home End", "↑↓") don't fit
// key.Binding cleanly, and the overlay's descriptions can be richer than the terse bar
// hints. Adding a binding here is the cost of having the overlay read as documentation.
func helpSections() []helpSection {
	return []helpSection{
		{"Global", []helpBinding{
			{"q", "quit"},
			{"t", "toggle status bar"},
			{"h", "this help"},
			{"↑↓", "scroll up/down"},
			{"←→", "scroll left/right"},
			{"PgUp PgDn", "page up/down"},
			{"Home End", "top/bottom"},
			{"Shift+←→", "page horizontal"},
		}},
		{"View", []helpBinding{
			{"d", "toggle diff"},
			{"p", "pause"},
			{"r", "record"},
			{"/", "search"},
			{"b", "history"},
			{"Esc", "jump to live tail"},
		}},
		{"Picker", []helpBinding{
			{"←→", "frame ±1"},
			{"Home End", "first/last"},
			{"Enter b", "confirm"},
			{"Esc", "back to view"},
		}},
		{"Search", []helpBinding{
			{"n", "next match"},
			{"N", "prev match"},
			{"/", "new search"},
			{"Esc", "back"},
		}},
		{"Input", []helpBinding{
			{"Enter", "submit"},
			{"Esc", "cancel"},
		}},
	}
}

// renderHelpPanel composes the centered help overlay: a two-column layout (Global+View on
// the left, Picker+Search+Input on the right) wrapped in a rounded border with no fill. A
// dismiss tip ("h to close") is embedded in the bottom border, centered, so it doesn't eat
// vertical space inside the panel.
func renderHelpPanel() string {
	sections := helpSections()
	body := lipgloss.JoinHorizontal(lipgloss.Top,
		renderHelpColumn(sections[:2]),
		helpColumnGap,
		renderHelpColumn(sections[2:]),
	)
	return embedBottomBorderTip(helpPanelStyle.Render(body), "h to close")
}

// embedBottomBorderTip rewrites the panel's last (rounded-border) line, replacing its
// horizontal-dash run with `<tip>` centred between the corners. Falls back to the
// unmodified panel if the tip can't fit with at least one dash on each side.
func embedBottomBorderTip(panel, tip string) string {
	lines := strings.Split(panel, "\n")
	if len(lines) == 0 {
		return panel
	}
	bottom := lines[len(lines)-1]
	bottomW := lipgloss.Width(bottom)
	// Need room for: 2 corners + 1-cell pad each side of the tip.
	if bottomW < lipgloss.Width(tip)+4 {
		return panel
	}
	paddedTip := lipgloss.NewStyle().Padding(0, 1).Render(tip)
	inner := lipgloss.PlaceHorizontal(bottomW-2, lipgloss.Center, paddedTip,
		lipgloss.WithWhitespaceChars("─"))
	lines[len(lines)-1] = "╰" + inner + "╯"
	return strings.Join(lines, "\n")
}

// renderHelpColumn renders a vertical stack of sections as "<bold key>   <desc>" rows,
// keys padded to the widest key in the column so descriptions align.
func renderHelpColumn(sections []helpSection) string {
	keyWidth := 0
	for _, s := range sections {
		for _, b := range s.bindings {
			keyWidth = max(keyWidth, lipgloss.Width(b.keys))
		}
	}
	keyCell := lipgloss.NewStyle().Bold(true).Width(keyWidth)

	var lines []string
	for i, s := range sections {
		if i > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, s.title)
		for _, b := range s.bindings {
			lines = append(lines, fmt.Sprintf("  %s   %s", keyCell.Render(b.keys), b.desc))
		}
	}
	return strings.Join(lines, "\n")
}
