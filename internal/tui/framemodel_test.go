package tui

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/ivoronin/wch/internal/session"
)

// A new row inserted at the top while every row's age ticks must keep the mid-screen row
// pinned under the top edge (the core fix).
func TestAnchorKeepsRowOnPrepend(t *testing.T) {
	old := podNames(20)
	updated := append([]string{"pod-00"}, old...)

	m := newSizedModel(t, podTable("5m", old))
	m.frames.SetYOffset(5)
	if m.frames.YOffset() != 5 {
		t.Fatalf("setup offset=%d want 5", m.frames.YOffset())
	}

	m = feed(t, m, execResultMsg{exec: session.Execution{Stdout: podTable("6m", updated)}})
	if got := m.frames.YOffset(); got != 6 {
		t.Errorf("anchored offset=%d want 6 (pod-05 shifted down by inserted pod-00)", got)
	}
}

// At the very top, newly prepended rows stay visible (sticky top wins over the anchor).
func TestStickyTop(t *testing.T) {
	old := podNames(20)
	updated := append([]string{"pod-00"}, old...)

	m := newSizedModel(t, podTable("5m", old))
	if m.frames.YOffset() != 0 {
		t.Fatalf("setup offset=%d want 0", m.frames.YOffset())
	}

	m = feed(t, m, execResultMsg{exec: session.Execution{Stdout: podTable("6m", updated)}})
	if got := m.frames.YOffset(); got != 0 {
		t.Errorf("sticky top offset=%d want 0", got)
	}
}

// While viewing a past frame, its highlights are intrinsic (diff vs the previous recorded
// frame) and must survive a command run.
func TestHistoryHighlightsSurviveRefresh(t *testing.T) {
	names := podNames(3)
	m := New(Config{Command: "x", Interval: time.Second, DiffEnabled: true})
	m = feed(t, m, tea.WindowSizeMsg{Width: 40, Height: 10})
	m = feed(t, m, execResultMsg{exec: session.Execution{Stdout: podTable("5m", names)}})
	m = feed(t, m, execResultMsg{exec: session.Execution{Stdout: podTable("6m", names)}})
	m = feed(t, m, execResultMsg{exec: session.Execution{Stdout: podTable("7m", names)}})

	m = m.withCursor(1)
	if m.isFollowing() {
		t.Fatal("setup: expected to be viewing a past frame, not following")
	}

	plain := m
	plain.prefs.Diff = false
	before := m.frames.Frame(1, m.prefs.Diff)
	if before == plain.frames.Frame(1, plain.prefs.Diff) {
		t.Fatalf("frame 1 should be highlighted (diff vs frame 0); body=%q", before)
	}

	idx, off := m.cursor.Index(), m.frames.YOffset()
	m = feed(t, m, execResultMsg{exec: session.Execution{Stdout: podTable("8m", names)}})

	if m.cursor.Index() != idx {
		t.Errorf("historyIndex moved to %d, want %d (frozen while viewing past)", m.cursor.Index(), idx)
	}
	if got := m.frames.YOffset(); got != off {
		t.Errorf("YOffset moved to %d, want %d", got, off)
	}
	if after := m.frames.Frame(1, m.prefs.Diff); after != before {
		t.Errorf("highlights reset after refresh:\n before=%q\n after =%q", before, after)
	}
}

// At the very bottom, the view follows the tail on refresh.
func TestStickyBottom(t *testing.T) {
	old := podNames(20)
	updated := append(podNames(20), "pod-21")

	m := newSizedModel(t, podTable("5m", old))
	m.frames.GotoBottom()
	if !m.frames.AtBottom() {
		t.Fatalf("setup: expected to start at bottom")
	}

	m = feed(t, m, execResultMsg{exec: session.Execution{Stdout: podTable("6m", updated)}})
	if !m.frames.AtBottom() {
		t.Errorf("sticky bottom: expected to follow the tail, offset=%d", m.frames.YOffset())
	}
}
