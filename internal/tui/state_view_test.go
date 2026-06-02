package tui

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/ivoronin/wch/internal/session"
)

// TestViewStateBodyReturnsFrame: viewState.Body returns the frame at historyIndex,
// matching frames.Frame(historyIndex, prefs.Diff).
func TestViewStateBodyReturnsFrame(t *testing.T) {
	m := makePaintModel(t)
	m.state = viewState{}

	body, ok := (viewState{}).Body(m)
	if !ok {
		t.Fatalf("viewState.Body: ok=false with history present")
	}
	if want := m.frames.Frame(m.cursor.Index(), m.prefs.Diff); body != want {
		t.Errorf("viewState.Body returned different body than frames.Frame")
	}
}

// TestViewStateBodyEmptyHistory: viewState.Body returns ok=false before any
// frame is recorded, so the orchestrator leaves the viewport untouched.
func TestViewStateBodyEmptyHistory(t *testing.T) {
	m := New(Config{Command: "x", Interval: time.Second})
	if _, ok := (viewState{}).Body(m); ok {
		t.Errorf("viewState.Body: ok=true with empty history")
	}
}

func TestViewStateShowsBar(t *testing.T) {
	if got := (viewState{}).ShowsBar(); got != false {
		t.Errorf("viewState.ShowsBar() = %v, want false", got)
	}
}

func TestViewStateIsFrozen(t *testing.T) {
	if got := (viewState{}).IsFrozen(); got != false {
		t.Errorf("viewState.IsFrozen() = %v, want false", got)
	}
}

func TestViewStateFollowsTail(t *testing.T) {
	cases := []struct {
		name      string
		wasAtTail bool
		want      bool
	}{
		{"at tail", true, true},
		{"not at tail", false, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := (viewState{}).FollowsTail(c.wasAtTail); got != c.want {
				t.Errorf("FollowsTail(%v) = %v, want %v", c.wasAtTail, got, c.want)
			}
		})
	}
}

// TestViewStateHandlePicker pins that 'b' from viewState enters pickerState.
func TestViewStateHandlePicker(t *testing.T) {
	m := makePaintModel(t)
	m.state = viewState{}
	_, st, _, handled := viewState{}.Handle(m, tea.KeyPressMsg{Code: 'b', Text: "b"})
	if !handled {
		t.Fatalf("viewState.Handle('b') reported handled=false")
	}
	if _, ok := st.(pickerState); !ok {
		t.Errorf("viewState.Handle('b') returned %T, want pickerState", st)
	}
}

// viewHelpBindings swaps the first slot based on whether the user is following the tail
// (b history) or viewing a past frame (Esc back). Help and Quit are always present.
func TestViewHelpBindingsSwapsFirstSlotByFollowing(t *testing.T) {
	m := New(Config{Command: "x", Interval: time.Second})
	m = feed(t, m, tea.WindowSizeMsg{Width: 40, Height: 10})
	m = feed(t, m, execResultMsg{exec: session.Execution{
		Timestamp: time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC), Stdout: "alpha\n",
	}})
	m = feed(t, m, execResultMsg{exec: session.Execution{
		Timestamp: time.Date(2026, 5, 30, 12, 0, 1, 0, time.UTC), Stdout: "beta\n",
	}})

	if !m.isFollowing() {
		t.Fatalf("setup: should be following")
	}
	bindings := viewHelpBindings(m)
	if got := len(bindings); got != 3 {
		t.Errorf("at tail: viewHelpBindings len = %d, want 3 (Picker, Help, Quit)", got)
	}
	if got := bindings[0].Help().Key; got != viewKeys.Picker.Help().Key {
		t.Errorf("at tail: first binding key = %q, want %q", got, viewKeys.Picker.Help().Key)
	}

	m = m.withCursor(0) // view first frame
	if m.isFollowing() {
		t.Fatalf("setup: should be viewing past")
	}
	bindings = viewHelpBindings(m)
	if got := len(bindings); got != 3 {
		t.Errorf("past frame: viewHelpBindings len = %d, want 3 (Esc, Help, Quit)", got)
	}
	if got := bindings[0].Help().Key; got != commonKeys.Escape.Help().Key {
		t.Errorf("past frame: first binding key = %q, want %q", got, commonKeys.Escape.Help().Key)
	}
}
