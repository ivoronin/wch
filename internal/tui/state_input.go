package tui

import (
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/ivoronin/wch/internal/tui/notify"
)

// inputBarStyles is the default textinput palette for any feature that wants its prompt to
// live inside the bottom status bar: every sub-element carries the bar's fg/bg so a
// long-value scroll-overflow doesn't bleed unstyled cells. Placeholder and suggestion inherit
// the bar plate and dim via Faint. The cursor is an explicit Cyan - visible against both
// light and dark bar plates.
var inputBarStyles = func() textinput.Styles {
	on := barInnerStyle
	dim := barInnerStyle.Faint(true)
	state := textinput.StyleState{
		Text:        on,
		Prompt:      on,
		Placeholder: dim,
		Suggestion:  dim,
	}
	s := textinput.DefaultDarkStyles()
	s.Focused = state
	s.Blurred = state
	s.Cursor.Color = ansi.Cyan
	return s
}()

// inputKeys gates the universal input bindings (submit/cancel) shown in every input flow.
var inputKeys = struct {
	Submit key.Binding
	Cancel key.Binding
}{
	Submit: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "save")),
	Cancel: key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
}

// Body for inputState delegates to its host: the input prompt lives in the bar; the
// viewport content belongs to whatever state input is sitting on top of.
func (s inputState) Body(m Model) (string, bool) {
	return s.prev.Body(m)
}

// Timestamp delegates to the host: input is transparent for bar-clock concerns.
func (s inputState) Timestamp(m Model) (time.Time, bool) {
	return s.prev.Timestamp(m)
}

// ShowsBar returns true because the input prompt lives in the bar; hiding it
// would leave the user typing into nothing. The decision does not depend on
// prev — input always wants its own bar.
func (inputState) ShowsBar() bool { return true }

// IsFrozen delegates to prev: inputState is transparent for viewport-derived
// concerns, so freezing semantics belong to the host.
func (s inputState) IsFrozen() bool { return s.prev.IsFrozen() }

// FollowsTail delegates to prev for the same reason: tail-follow policy
// belongs to the host state, not the input prompt overlaid on top.
func (s inputState) FollowsTail(wasAtTail bool) bool {
	return s.prev.FollowsTail(wasAtTail)
}

// RenderBar renders the input widget across the full bar width, leaving room
// for the prompt and the bar's own horizontal padding so long values scroll
// inside the field instead of pushing the prompt off the left edge.
func (s inputState) RenderBar(m Model) string {
	w := m.width - statusBarStyle.GetHorizontalFrameSize() - lipgloss.Width(s.input.Prompt)
	if w > 0 {
		s.input.SetWidth(w)
	}
	return statusBarStyle.Width(m.width).Render(s.input.View())
}

// Handle routes a key in inputState. Submit calls the configured submit
// function directly (no message hop). Cancel pops back to prev. Every other
// key is forwarded to the textinput; handled=true on every key keeps globals
// (q/t) from stealing printable chars.
func (s inputState) Handle(m Model, msg tea.KeyPressMsg) (Model, state, tea.Cmd, bool) {
	switch {
	case key.Matches(msg, inputKeys.Submit):
		m2, st, cmd := s.submit(m, s)
		return m2, st, cmd, true
	case key.Matches(msg, inputKeys.Cancel):
		return m, s.prev, nil, true
	}
	in, cmd := s.input.Update(msg)
	s.input = in
	return m, s, cmd, true
}

// openInput is the shared constructor for the two input flows: builds an inputState wrapping
// the supplied textinput + submit callback on top of prev, and returns the textinput's focus
// Cmd. Order matters: textinput.Focus has a pointer receiver, so it must run BEFORE the
// inputState is built — otherwise the struct captures an unfocused snapshot of `in` and
// textinput.Update drops every keystroke.
func (m Model) openInput(prev state, in textinput.Model, submit func(Model, inputState) (Model, state, tea.Cmd)) (Model, state, tea.Cmd) {
	focusCmd := in.Focus()
	return m, inputState{input: in, prev: prev, submit: submit}, focusCmd
}

// openSearchInput builds the inputState for a search prompt. If we are already inside an
// active search (the user pressed '/' to start over), input.prev skips the abandoned search
// so Esc from the replacement returns to the pre-search predecessor.
func (m Model) openSearchInput(from state) (Model, state, tea.Cmd) {
	prev := from
	if s, ok := from.(searchState); ok {
		prev = s.prev
	}
	return m.openInput(prev, newBarInput(searchPromptLabel), applySearchSubmit)
}

// applySearchSubmit decides what to do with the typed query: empty → silent pop; no matches
// → notification + pop; otherwise → enter a fresh searchState with the captured body and
// matches, applying the selection overlay + scroll-to-match.
func applySearchSubmit(m Model, s inputState) (Model, state, tea.Cmd) {
	q := strings.TrimSpace(s.input.Value())
	if q == "" {
		return m, s.prev, nil
	}
	// Capture the underlying state's body. After openSearchInput's prev-peel,
	// s.prev is always view/picker — never searchState — so the body is just
	// the current frame. Frame returns "" for an invalid index, so the no-cursor
	// case (no history yet) flows into the "no matches" branch below without a
	// separate guard.
	i, ok := m.cursor.At()
	body := m.frames.Frame(i, m.prefs.Diff)
	matches := findMatches(ansi.Strip(body), q)
	if len(matches) == 0 {
		var cmd tea.Cmd
		m, cmd = m.push(notify.LevelInfo, "no matches")
		return m, s.prev, cmd
	}
	next := searchState{
		query:    q,
		body:     body,
		matches:  matches,
		selected: 0,
		follow:   m.isFollowing(),
		prev:     s.prev,
	}
	if ok {
		next.captured = m.session.History[i].Timestamp
	}
	m = m.repaintWith(next, next.snap())
	return m, next, nil
}

// newBarInput constructs a textinput pre-styled to live in the bottom bar.
func newBarInput(prompt string) textinput.Model {
	in := textinput.New()
	in.Prompt = prompt
	in.SetStyles(inputBarStyles)
	return in
}
