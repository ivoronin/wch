package tui

import (
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/compat"
	"github.com/charmbracelet/x/ansi"
)

// insertFg is the foreground the diff renderer applies to changed cells.
var insertFg ansi.Color = ansi.Green

// Layout constants
const (
	itemSpacing  = 2          // visual gap between picker timestamps (matches "  ")
	arrowWidth   = 2          // width of "◀ " or " ▶" navigation arrows
	timestampFmt = "15:04:05" // HH:MM:SS format for history timestamps
)

var (
	// barBg / barFg are the only AdaptiveColor pair the bar surface needs. Plates use these;
	// accents on top use named ANSI so the terminal's palette resolves them per theme.
	barBg = compat.AdaptiveColor{
		Light: lipgloss.Color("#e5e5e5"),
		Dark:  lipgloss.Color("#262626"),
	}
	barFg = compat.AdaptiveColor{
		Light: lipgloss.Color("#1f1f1f"),
		Dark:  lipgloss.Color("#d6d6d6"),
	}

	statusBarStyle = lipgloss.NewStyle().
			Background(barBg).
			Foreground(barFg).
			Padding(0, 1)

	// barInnerStyle is the fg/bg every inline bar element carries so its own SGR Reset
	// doesn't tear the plate. Each element re-opens this style.
	barInnerStyle = lipgloss.NewStyle().
			Background(barBg).
			Foreground(barFg)

	helpStyle = lipgloss.NewStyle().
			PaddingLeft(1).
			Background(barBg).
			Foreground(barFg)

	// helpColumnGap separates the two columns inside the help overlay panel.
	helpColumnGap = lipgloss.NewStyle().Width(4).Render("")

	helpPanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(1, 2)

	// indicatorStyle must render exactly 1 visible cell — centerBlockWidth assumes 1-cell
	// slots with gaps emitted explicitly by renderCenterBlock. Padding here would push the
	// bar past contentWidth and wrap the help text onto a second row.
	indicatorStyle = lipgloss.NewStyle().
			Background(barBg).
			Foreground(barFg)

	errorStyle = lipgloss.NewStyle().
			Foreground(ansi.Red)

	// recStyle: same 1-cell invariant as indicatorStyle. Red dot on the bar's own plate, not
	// a red slab — the glyph itself signals recording, the surrounding bg stays consistent.
	recStyle = lipgloss.NewStyle().
			Background(barBg).
			Foreground(ansi.Red)

	pickerItemStyle = lipgloss.NewStyle().
			Background(barBg).
			Foreground(barFg)

	// pickerSelectedStyle: yellow plate, black text — distinguishes the active timestamp
	// from the surrounding bar plate.
	pickerSelectedStyle = lipgloss.NewStyle().
				Background(ansi.Yellow).
				Foreground(ansi.Black)
)

// boldKeep wraps s in a bold span that terminates with SGR 22 (bold-off) instead of the
// full reset that lipgloss.Render emits. Use inside a styled outer span (e.g. helpStyle)
// where a full reset would erase the outer fg/bg for the tail of the joined string.
func boldKeep(s string) string {
	return "\x1b[1m" + s + "\x1b[22m"
}
