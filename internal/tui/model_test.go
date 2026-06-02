package tui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

// NewReplay never schedules a tick and renders the latest frame after the first WindowSizeMsg.
// Init may return a non-nil cmd (e.g. background-color query) but must never schedule a tick.
func TestNewReplayInitNoTickAndRendersLatest(t *testing.T) {
	s := preloadedReplaySession(3)
	m := NewReplay(Config{}, s)

	if cmd := m.Init(); cmd != nil {
		if msg := cmd(); msg != nil {
			if _, isTick := msg.(tickMsg); isTick {
				t.Errorf("replay Init() must not schedule a tick")
			}
		}
	}
	if m.isLive() {
		t.Errorf("replay isLive() should be false")
	}
	if m.cursor.Index() != 2 {
		t.Errorf("historyIndex=%d want 2 (last frame)", m.cursor.Index())
	}

	m = feed(t, m, tea.WindowSizeMsg{Width: 40, Height: 10})
	if !m.ready {
		t.Errorf("Model.ready should be true after WindowSizeMsg")
	}
	body := m.frames.Frame(m.cursor.Index(), m.prefs.Diff)
	if !strings.Contains(body, "frame 2") {
		t.Errorf("expected latest frame rendered; got %q", body)
	}
}

// 'q' falls through from viewState's Handle to handleGlobalKey for the Quit cmd.
func TestHandleKeyPriorityFallsThroughToGlobal(t *testing.T) {
	m := New(Config{Command: "x", Interval: time.Second})
	m = feed(t, m, tea.WindowSizeMsg{Width: 40, Height: 10})

	_, cmd := m.dispatchKey(tea.KeyPressMsg{Code: 'q', Text: "q"})
	if cmd == nil {
		t.Fatalf("expected tea.Quit cmd from q-key fallthrough")
	}
	if msg := cmd(); msg != (tea.QuitMsg{}) {
		t.Errorf("q-key cmd produced %T (%v), want tea.QuitMsg{}", msg, msg)
	}
}
