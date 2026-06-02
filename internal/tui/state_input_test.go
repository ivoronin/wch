package tui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

// In inputState, a bracketed-paste msg should land in the textinput's value. Update must
// forward tea.PasteMsg to the active state — otherwise terminal pastes silently vanish.
func TestInputStateReceivesPasteMsg(t *testing.T) {
	m := New(Config{Command: "kubectl", Interval: time.Second})
	m = feed(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})
	m = pressKey(t, m, 'r')
	in, ok := m.state.(inputState)
	if !ok {
		t.Fatalf("setup: state = %T, want inputState", m.state)
	}
	in.input.SetValue("")
	m.state = in

	m = feed(t, m, tea.PasteMsg{Content: "hello"})

	got := m.state.(inputState).input.Value()
	if !strings.Contains(got, "hello") {
		t.Errorf("input value after paste = %q, want it to contain \"hello\"", got)
	}
}

// TestInputStateBodyDelegatesToPrev: inputState is transparent; its Body returns
// exactly what prev.Body returns.
func TestInputStateBodyDelegatesToPrev(t *testing.T) {
	m := makePaintModel(t)
	prev := viewState{}
	in := inputState{prev: prev}

	prevBody, prevOk := prev.Body(m)
	gotBody, gotOk := in.Body(m)
	if gotOk != prevOk {
		t.Errorf("inputState.Body ok=%v, want %v", gotOk, prevOk)
	}
	if gotBody != prevBody {
		t.Errorf("inputState.Body did not delegate to prev")
	}
}

func TestInputStateShowsBar(t *testing.T) {
	cases := []struct {
		name string
		prev state
	}{
		{"on view", viewState{}},
		{"on picker", pickerState{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := inputState{prev: c.prev}
			if !s.ShowsBar() {
				t.Errorf("inputState.ShowsBar() = false, want true")
			}
		})
	}
}

func TestInputStateIsFrozenDelegatesToPrev(t *testing.T) {
	cases := []struct {
		name string
		prev state
		want bool
	}{
		{"on view", viewState{}, false},
		{"on picker", pickerState{}, false},
		{"on search", searchState{prev: viewState{}}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := (inputState{prev: c.prev}).IsFrozen(); got != c.want {
				t.Errorf("IsFrozen() = %v, want %v", got, c.want)
			}
		})
	}
}

func TestInputStateFollowsTailDelegatesToPrev(t *testing.T) {
	cases := []struct {
		name      string
		prev      state
		wasAtTail bool
		want      bool
	}{
		{"on view at tail", viewState{}, true, true},
		{"on view not at tail", viewState{}, false, false},
		{"on search inherits search follow", searchState{prev: viewState{}, follow: true}, false, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := (inputState{prev: c.prev}).FollowsTail(c.wasAtTail); got != c.want {
				t.Errorf("FollowsTail(%v) = %v, want %v", c.wasAtTail, got, c.want)
			}
		})
	}
}

// TestInputStateHandleEscPopsToPrev pins that Esc returns to the wrapped prev state.
func TestInputStateHandleEscPopsToPrev(t *testing.T) {
	m := makePaintModel(t)
	in := inputState{prev: viewState{}, input: newBarInput("/")}
	m.state = in
	_, st, _, handled := in.Handle(m, tea.KeyPressMsg{Code: tea.KeyEsc})
	if !handled {
		t.Fatalf("inputState.Handle(Esc) reported handled=false")
	}
	if _, ok := st.(viewState); !ok {
		t.Errorf("inputState.Handle(Esc) returned %T, want viewState (the prev)", st)
	}
}
