package notify

import (
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// defaultStyles returns the per-level styles. Info uses the terminal's default colors
// (no tint); Warning gets a yellow-tinted border. The level symbol itself (ℹ / ⚠) is the
// only accented piece — see prefixFor.
func defaultStyles() map[Level]lipgloss.Style {
	base := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1)
	return map[Level]lipgloss.Style{
		LevelInfo:    base,
		LevelWarning: base.BorderForeground(ansi.Yellow),
	}
}

// Level symbol accents: a small splash of color on the leading glyph so the level reads at
// a glance without tinting the rest of the bubble.
var (
	infoAccent    = lipgloss.NewStyle().Foreground(ansi.Cyan)
	warningAccent = lipgloss.NewStyle().Foreground(ansi.Yellow)
)

// prefixFor returns the level-specific icon prefix shown before the message, with its
// accent color baked in.
func prefixFor(l Level) string {
	switch l {
	case LevelWarning:
		return warningAccent.Render("⚠ ")
	default:
		return infoAccent.Render("ℹ ")
	}
}
