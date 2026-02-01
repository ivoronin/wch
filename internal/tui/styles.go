package tui

import "github.com/charmbracelet/lipgloss"

var (
	statusBarStyle = lipgloss.NewStyle().
			Faint(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF5555")).
			Bold(true)
)
