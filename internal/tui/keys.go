package tui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Quit       key.Binding
	Up         key.Binding
	Down       key.Binding
	Left       key.Binding
	Right      key.Binding
	PageUp     key.Binding
	PageDown   key.Binding
	Home       key.Binding
	End        key.Binding
	ToggleDiff key.Binding
	ToggleBar  key.Binding
	Pause      key.Binding
}

func newKeyMap() keyMap {
	return keyMap{
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
		),
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
		),
		Left: key.NewBinding(
			key.WithKeys("left", "h"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", "l"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown"),
		),
		Home: key.NewBinding(
			key.WithKeys("home", "g"),
		),
		End: key.NewBinding(
			key.WithKeys("end", "G"),
		),
		ToggleDiff: key.NewBinding(
			key.WithKeys("d"),
		),
		ToggleBar: key.NewBinding(
			key.WithKeys("t"),
		),
		Pause: key.NewBinding(
			key.WithKeys("p"),
		),
	}
}
