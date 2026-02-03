package tui

import "github.com/charmbracelet/lipgloss"

// threeColumnLayout holds computed widths for a three-section horizontal layout.
type threeColumnLayout struct {
	leftWidth  int // width for left section
	rightWidth int // width for right section
}

// calcThreeColumnLayout computes widths for a three-section layout where
// the center section is centered within the content area.
// totalWidth is the full available width, centerWidth is the width of the centered content.
func calcThreeColumnLayout(totalWidth, centerWidth int) threeColumnLayout {
	contentWidth := totalWidth - statusBarStyle.GetHorizontalFrameSize()
	leftWidth := (contentWidth - centerWidth) / 2
	rightWidth := contentWidth - leftWidth - centerWidth
	return threeColumnLayout{
		leftWidth:  leftWidth,
		rightWidth: rightWidth,
	}
}

// renderLeft renders content left-aligned within the given width.
func renderLeft(content string, width int) string {
	return lipgloss.NewStyle().Width(width).Render(content)
}

// renderRight renders content right-aligned within the given width.
func renderRight(content string, width int) string {
	return lipgloss.NewStyle().Width(width).Align(lipgloss.Right).Render(content)
}
