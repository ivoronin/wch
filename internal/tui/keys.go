package tui

import "github.com/charmbracelet/bubbles/key"

// Shared navigation key bindings used across multiple modes.
var navKeys = struct {
	Left  key.Binding
	Right key.Binding
}{
	Left:  key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("←/h", "left")),
	Right: key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("→/l", "right")),
}

// Global key bindings (work in all modes)
var globalKeys = struct {
	Quit       key.Binding
	Escape     key.Binding
	ToggleDiff key.Binding
	ToggleBar  key.Binding
	Pause      key.Binding
}{
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Escape: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back"),
	),
	ToggleDiff: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "diff"),
	),
	ToggleBar: key.NewBinding(
		key.WithKeys("t"),
		key.WithHelp("t", "status"),
	),
	Pause: key.NewBinding(
		key.WithKeys("p"),
		key.WithHelp("p", "pause"),
	),
}

// View mode key bindings (mode-specific only; uses shared navKeys for left/right)
var viewKeys = struct {
	Up          key.Binding
	Down        key.Binding
	PageUp      key.Binding
	PageDown    key.Binding
	ScrollLeft  key.Binding
	ScrollRight key.Binding
	Home        key.Binding
	End         key.Binding
	Picker      key.Binding
}{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	PageUp: key.NewBinding(
		key.WithKeys("pgup"),
		key.WithHelp("pgup", "page up"),
	),
	PageDown: key.NewBinding(
		key.WithKeys("pgdown"),
		key.WithHelp("pgdn", "page down"),
	),
	ScrollLeft:  key.NewBinding(key.WithKeys("shift+left")),
	ScrollRight: key.NewBinding(key.WithKeys("shift+right")),
	Home: key.NewBinding(
		key.WithKeys("home", "g"),
		key.WithHelp("home", "top"),
	),
	End: key.NewBinding(
		key.WithKeys("end", "G"),
		key.WithHelp("end", "bottom"),
	),
	Picker: key.NewBinding(
		key.WithKeys("b"),
		key.WithHelp("b", "browse"),
	),
}

// Picker mode key bindings (mode-specific only; uses shared navKeys for left/right)
var pickerKeys = struct {
	Confirm key.Binding
}{
	Confirm: key.NewBinding(key.WithKeys("enter", "b"), key.WithHelp("enter", "confirm")),
}
