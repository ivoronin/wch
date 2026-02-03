package tui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// Mode defines behavior for a UI interaction mode.
type Mode interface {
	// ShortHelp returns key bindings for help display.
	ShortHelp() []key.Binding

	// HandleKey processes mode-specific input.
	// Returns a message describing the action to take, or nil if unhandled.
	HandleKey(msg tea.KeyMsg) tea.Msg
}

// Message types are defined in messages.go
// Key bindings are defined in keys.go

// handleGlobalKey processes keys that work in all modes.
// Returns a message describing the action to take, or nil if unhandled.
func handleGlobalKey(msg tea.KeyMsg) tea.Msg {
	switch {
	case key.Matches(msg, globalKeys.Quit):
		return tea.QuitMsg{}
	case key.Matches(msg, globalKeys.Escape):
		return escapeMsg{}
	case key.Matches(msg, globalKeys.ToggleDiff):
		return toggleDiffMsg{}
	case key.Matches(msg, globalKeys.ToggleBar):
		return toggleBarMsg{}
	case key.Matches(msg, globalKeys.Pause):
		return togglePauseMsg{}
	}
	return nil
}
