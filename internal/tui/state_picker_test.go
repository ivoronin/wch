package tui

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/ivoronin/wch/internal/session"
)

// pickerState.Handle maps cursor keys to historyIndex movements and Confirm/Esc to a state
// return of viewState{}. Unhandled keys (e.g. arbitrary letters) report handled=false so
// the global fallback can fire.
func TestHandlePickerKey(t *testing.T) {
	// Need a model with history so withCursor doesn't clamp to -1.
	m := New(Config{Command: "x", Interval: time.Second})
	m = feed(t, m, tea.WindowSizeMsg{Width: 40, Height: 10})
	for i := 0; i < 5; i++ {
		m = feed(t, m, execResultMsg{exec: session.Execution{
			Timestamp: time.Date(2026, 5, 30, 12, 0, i, 0, time.UTC),
			Stdout:    fmt.Sprintf("frame %d\n", i),
		}})
	}

	cases := []struct {
		name      string
		key       tea.KeyPressMsg
		wantState string // type name, checked via type switch
		wantIndex int
	}{
		{"left", tea.KeyPressMsg{Code: tea.KeyLeft}, "tui.pickerState", 3},
		{"right (clamped)", tea.KeyPressMsg{Code: tea.KeyRight}, "tui.pickerState", 4},
		{"confirm", tea.KeyPressMsg{Code: tea.KeyEnter}, "tui.viewState", 4},
		{"esc", tea.KeyPressMsg{Code: tea.KeyEsc}, "tui.viewState", 4},
	}

	for _, c := range cases {
		// historyIndex starts at 4 (tail).
		mm := m
		mm.state = pickerState{}
		mm.cursor = cursorAt(4)
		newM, s, _, handled := pickerState{}.Handle(mm, c.key)
		if !handled {
			t.Errorf("%s: handled=false, want true", c.name)
		}
		got := reflect.TypeOf(s).String()
		if got != c.wantState {
			t.Errorf("%s: state = %s, want %s", c.name, got, c.wantState)
		}
		if newM.cursor.Index() != c.wantIndex {
			t.Errorf("%s: historyIndex = %d, want %d", c.name, newM.cursor.Index(), c.wantIndex)
		}
	}

	// Unhandled key.
	_, _, _, handled := pickerState{}.Handle(m, tea.KeyPressMsg{Code: 'x', Text: "x"})
	if handled {
		t.Errorf("unhandled key reported handled=true")
	}
}

// An empty history yields a blank, width-sized bar (no panic, no out-of-range access).
func TestRenderPickerTimelineEmpty(t *testing.T) {
	if got, want := renderPickerTimeline(nil, 0, 40), statusBarStyle.Width(40).Render(""); got != want {
		t.Errorf("renderPickerTimeline(empty)=%q want %q", got, want)
	}
}

// The bar shows the selected timestamp plus its neighbours and fits the given width.
func TestRenderPickerTimelineShowsTimestamps(t *testing.T) {
	const width = 80
	base := time.Date(2026, 5, 29, 12, 0, 0, 0, time.UTC)
	history := make([]session.Execution, 5)
	for i := range history {
		history[i] = session.Execution{Timestamp: base.Add(time.Duration(i) * time.Second)}
	}

	out := renderPickerTimeline(history, 2, width)

	if w := lipgloss.Width(out); w != width {
		t.Errorf("rendered width=%d want %d", w, width)
	}
	plain := ansi.Strip(out)
	for i, exec := range history {
		ts := exec.Timestamp.Format(timestampFmt)
		if !strings.Contains(plain, ts) {
			t.Errorf("bar missing timestamp %d (%q): %q", i, ts, plain)
		}
	}
}

// TestPickerStateBodyReturnsFrame: picker renders the same body as view -- the
// timeline lives in the bar, not the viewport.
func TestPickerStateBodyReturnsFrame(t *testing.T) {
	m := makePaintModel(t)
	m.state = pickerState{}

	body, ok := (pickerState{}).Body(m)
	if !ok {
		t.Fatalf("pickerState.Body: ok=false with history present")
	}
	if want := m.frames.Frame(m.cursor.Index(), m.prefs.Diff); body != want {
		t.Errorf("pickerState.Body returned different body than frames.Frame")
	}
}

func TestPickerStateShowsBar(t *testing.T) {
	if got := (pickerState{}).ShowsBar(); got != true {
		t.Errorf("pickerState.ShowsBar() = %v, want true", got)
	}
}

func TestPickerStateIsFrozen(t *testing.T) {
	if got := (pickerState{}).IsFrozen(); got != false {
		t.Errorf("pickerState.IsFrozen() = %v, want false", got)
	}
}

func TestPickerStateFollowsTail(t *testing.T) {
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
			if got := (pickerState{}).FollowsTail(c.wasAtTail); got != c.want {
				t.Errorf("FollowsTail(%v) = %v, want %v", c.wasAtTail, got, c.want)
			}
		})
	}
}

// TestPickerStateHandleEscReturnsToView pins Esc in pickerState returns to viewState.
func TestPickerStateHandleEscReturnsToView(t *testing.T) {
	m := makePaintModel(t)
	m.state = pickerState{}
	_, st, _, handled := pickerState{}.Handle(m, tea.KeyPressMsg{Code: tea.KeyEsc})
	if !handled {
		t.Fatalf("pickerState.Handle(Esc) reported handled=false")
	}
	if _, ok := st.(viewState); !ok {
		t.Errorf("pickerState.Handle(Esc) returned %T, want viewState", st)
	}
}
