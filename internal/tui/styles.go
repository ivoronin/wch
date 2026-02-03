package tui

import "github.com/charmbracelet/lipgloss"

// Layout constants
const (
	itemSpacing  = 2          // visual gap between picker timestamps (matches "  ")
	arrowWidth   = 2          // width of "◀ " or " ▶" navigation arrows
	timestampFmt = "15:04:05" // HH:MM:SS format for history timestamps
)

// Color constants
const (
	colorDim    = lipgloss.Color("#888888")
	colorBright = lipgloss.Color("#FFFFFF")
	colorError  = lipgloss.Color("#FF5555")
	colorBg     = lipgloss.Color("#1e1e1e")
)

var (
	statusBarStyle = lipgloss.NewStyle().
			Foreground(colorDim).
			Background(colorBg).
			Padding(0, 1)

	helpStyle = lipgloss.NewStyle().PaddingLeft(1)

	indicatorStyle = lipgloss.NewStyle().PaddingLeft(1)

	errorStyle = lipgloss.NewStyle().
			Foreground(colorError).
			Bold(true)

	pickerItemStyle = lipgloss.NewStyle().
			Foreground(colorDim).
			Background(colorBg)

	pickerSelectedStyle = lipgloss.NewStyle().
				Foreground(colorBright).
				Background(colorBg).
				Bold(true)
)
