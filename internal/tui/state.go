package tui

import (
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

// viewPolicy is the display-side contract of a state: what to render in the viewport
// (Body), what timestamp to show in the bar's center clock (Timestamp), whether the bar
// shows unconditionally (ShowsBar), whether new executions should repaint (IsFrozen),
// how historyIndex advances (FollowsTail), and what to put in the bar (RenderBar).
// Display concerns never reach Handle. Body and Timestamp's (_, bool) shapes let a
// state opt out (e.g. viewState before the first frame, searchState with zero matches)
// without the orchestrator inventing a sentinel value.
type viewPolicy interface {
	Body(Model) (string, bool)
	Timestamp(Model) (time.Time, bool)
	ShowsBar() bool
	IsFrozen() bool
	FollowsTail(wasAtTail bool) bool
	RenderBar(Model) string
}

// inputHandler is the input-side contract of a state: a key press becomes a new model,
// the next state, and a side-effect cmd. handled gates the global-key fallback so an
// active overlay (input prompt, search) can consume printable chars before the global
// q/t bindings see them.
type inputHandler interface {
	Handle(Model, tea.KeyPressMsg) (Model, state, tea.Cmd, bool)
}

// state is the sealed sum type of UI modes. Exactly one struct below inhabits Model.state
// at any time; the isState marker keeps unrelated types out of the union. Display and
// input live behind the viewPolicy and inputHandler interfaces so a future reader can
// see the two concerns separately without scanning a 6-method blob.
type state interface {
	isState()
	viewPolicy
	inputHandler
}

// viewState is the default: the viewport shows the frame at Model.historyIndex, with diff
// highlights when Model.prefs.Diff is set. All other display fields are read directly off Model.
type viewState struct{}

// pickerState is the history-timeline picker. The cursor is Model.historyIndex; cursor
// movement reassigns it.
type pickerState struct{}

// inputState is a single-line prompt overlaid on top of a host state (prev). Used by both
// the record-filename and search-query flows; submit decides what to do with the typed
// value. inputState is transparent: Body/FollowsTail/IsFrozen all delegate to prev. The
// input prompt itself lives in the bar.
type inputState struct {
	input  textinput.Model
	prev   state
	submit func(Model, inputState) (Model, state, tea.Cmd)
}

// searchState shows a frozen snapshot with the selected match highlighted. body is captured
// once on entry; matches are computed once. follow remembers whether the user was at the
// live tail on entry so historyIndex can advance in the background — Esc then drops into
// the underlying viewState already at the new tail without a second keystroke.
type searchState struct {
	query    string
	body     string
	matches  []searchMatch
	selected int
	follow   bool
	captured time.Time
	prev     state
}

func (viewState) isState()   {}
func (pickerState) isState() {}
func (inputState) isState()  {}
func (searchState) isState() {}
