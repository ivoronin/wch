package tui

import (
	"slices"

	"charm.land/bubbles/v2/key"
)

var globalKeys = struct {
	Quit      key.Binding
	ToggleBar key.Binding
	Help      key.Binding
}{
	Quit:      key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	ToggleBar: key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "status")),
	Help:      key.NewBinding(key.WithKeys("h"), key.WithHelp("h", "help")),
}

// navKeys are the viewport-navigation defaults. handleGlobalKey scrolls for any of them
// not intercepted by the active state. pickerState intercepts Left/Right (frame cursor)
// and Home/End (first/last frame); inputState lets the textinput consume arrows for its
// own cursor.
var navKeys = struct {
	Left, Right             key.Binding
	Up, Down                key.Binding
	PageUp, PageDown        key.Binding
	Home, End               key.Binding
	ScrollLeft, ScrollRight key.Binding
}{
	Left:        key.NewBinding(key.WithKeys("left"), key.WithHelp("←", "left")),
	Right:       key.NewBinding(key.WithKeys("right"), key.WithHelp("→", "right")),
	Up:          key.NewBinding(key.WithKeys("up"), key.WithHelp("↑", "up")),
	Down:        key.NewBinding(key.WithKeys("down"), key.WithHelp("↓", "down")),
	PageUp:      key.NewBinding(key.WithKeys("pgup"), key.WithHelp("pgup", "page up")),
	PageDown:    key.NewBinding(key.WithKeys("pgdown"), key.WithHelp("pgdn", "page down")),
	Home:        key.NewBinding(key.WithKeys("home"), key.WithHelp("home", "top")),
	End:         key.NewBinding(key.WithKeys("end"), key.WithHelp("end", "bottom")),
	ScrollLeft:  key.NewBinding(key.WithKeys("shift+left")),
	ScrollRight: key.NewBinding(key.WithKeys("shift+right")),
}

// commonKeys are intercepted with identical semantics in both viewState and pickerState:
// toggle diff/pause, start/stop recording, open search. Held once to avoid duplicating
// the bindings (and the matching switch arms) across both handlers.
var commonKeys = struct {
	ToggleDiff key.Binding
	Pause      key.Binding
	Record     key.Binding
	Search     key.Binding
	Escape     key.Binding
}{
	ToggleDiff: key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "diff")),
	Pause:      key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "pause")),
	Record:     key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "record")),
	Search:     key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
	Escape:     key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
}

// viewKeys are viewState-specific bindings (entering the picker).
var viewKeys = struct {
	Picker key.Binding
}{
	Picker: key.NewBinding(key.WithKeys("b"), key.WithHelp("b", "history")),
}

// pickerKeys are pickerState-specific bindings (confirming a selection). Enter and the
// picker-entry key (b) are symmetric aliases.
var pickerKeys = struct {
	Confirm key.Binding
}{
	Confirm: key.NewBinding(key.WithKeys("enter", "b"), key.WithHelp("enter", "confirm")),
}

// searchKeys are searchState-specific bindings. n/p navigate matches. '/' and Esc reuse
// commonKeys.Search and commonKeys.Escape — same keys, same help, no point duplicating.
var searchKeys = struct {
	NavNext key.Binding
	NavPrev key.Binding
}{
	NavNext: key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "next")),
	NavPrev: key.NewBinding(key.WithKeys("N"), key.WithHelp("N", "prev")),
}

// minimalBarBindings is the canonical bottom-bar trailer: state-specific extras followed
// by the always-visible {Help, Quit} pair.
func minimalBarBindings(extras ...key.Binding) []key.Binding {
	return slices.Concat(extras, []key.Binding{globalKeys.Help, globalKeys.Quit})
}
