package tui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// viewMode is the default mode for viewing output.
type viewModeType struct{}

var viewMode Mode = viewModeType{}

// Key bindings are defined in keys.go

func (viewModeType) ShortHelp() []key.Binding {
	return []key.Binding{
		globalKeys.Quit,
		globalKeys.ToggleDiff,
		globalKeys.Pause,
		viewKeys.Picker,
	}
}

func (viewModeType) HandleKey(msg tea.KeyMsg) tea.Msg {
	switch {
	case key.Matches(msg, viewKeys.Picker):
		return switchModeMsg{mode: pickerMode}
	case key.Matches(msg, viewKeys.Up):
		return scrollUpMsg{lines: 1}
	case key.Matches(msg, viewKeys.Down):
		return scrollDownMsg{lines: 1}
	case key.Matches(msg, navKeys.Left):
		return scrollLeftMsg{}
	case key.Matches(msg, navKeys.Right):
		return scrollRightMsg{}
	case key.Matches(msg, viewKeys.PageUp):
		return pageUpMsg{}
	case key.Matches(msg, viewKeys.PageDown):
		return pageDownMsg{}
	case key.Matches(msg, viewKeys.ScrollLeft):
		return scrollLeftPageMsg{}
	case key.Matches(msg, viewKeys.ScrollRight):
		return scrollRightPageMsg{}
	case key.Matches(msg, viewKeys.Home):
		return gotoLeftEdgeMsg{}
	case key.Matches(msg, viewKeys.End):
		return gotoRightEdgeMsg{}
	}
	return nil
}
