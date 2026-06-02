package notify

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// Push schedules a Tick that, when fed back through Update, removes the bubble.
func TestPushExpireRoundTrip(t *testing.T) {
	m := New()
	var cmd tea.Cmd
	m, cmd = m.Push("hi", LevelInfo, 50*time.Millisecond)
	if !m.Active() {
		t.Fatalf("expected Active() == true after Push")
	}
	if cmd == nil {
		t.Fatalf("Push did not return a Cmd")
	}

	msg := cmd()
	exp, ok := msg.(expireMsg)
	if !ok {
		t.Fatalf("Tick produced %T, want expireMsg", msg)
	}

	m, _ = m.Update(exp)
	if m.Active() {
		t.Errorf("expected Active() == false after matching expireMsg")
	}
}

// A stale expireMsg (id no longer in the stack) is a harmless no-op — it must not wipe a
// newer bubble.
func TestStaleExpireIgnored(t *testing.T) {
	m := New()
	var firstCmd tea.Cmd
	m, firstCmd = m.Push("first", LevelInfo, time.Second)
	staleExp := firstCmd().(expireMsg)

	// Remove the first bubble explicitly, then push a newer one.
	m, _ = m.Update(staleExp)
	m, _ = m.Push("second", LevelInfo, time.Second)
	if !m.Active() {
		t.Fatalf("setup: second bubble should be active")
	}

	// Fire the stale expire again — must not affect the new bubble.
	m, _ = m.Update(staleExp)
	if !m.Active() {
		t.Errorf("stale expire wiped a newer bubble")
	}
}

// WithMaxVisible caps the stack; oldest bubbles are evicted on overflow. Stale expires for
// evicted ones are no-ops.
func TestMaxVisibleEviction(t *testing.T) {
	m := New(WithMaxVisible(2))
	var cmd1 tea.Cmd
	m, cmd1 = m.Push("one", LevelInfo, time.Second)
	m, _ = m.Push("two", LevelInfo, time.Second)
	m, _ = m.Push("three", LevelInfo, time.Second)

	if got := len(m.bubbles); got != 2 {
		t.Fatalf("len(bubbles)=%d, want 2 (cap)", got)
	}
	if m.bubbles[0].message != "two" || m.bubbles[1].message != "three" {
		t.Errorf("retained bubbles=%q,%q; want two,three (oldest evicted)",
			m.bubbles[0].message, m.bubbles[1].message)
	}
	// Fire the (now stale) expire of "one" — must not change the cap.
	m, _ = m.Update(cmd1().(expireMsg))
	if got := len(m.bubbles); got != 2 {
		t.Errorf("evicted-bubble expire mutated stack: len=%d, want 2", got)
	}
}

// Update returns the model untouched for foreign messages.
func TestUpdateForeignMessages(t *testing.T) {
	m := New()
	m, _ = m.Push("hi", LevelInfo, time.Second)
	wantLen := len(m.bubbles)
	for _, msg := range []tea.Msg{
		tea.WindowSizeMsg{Width: 80, Height: 24},
		tea.KeyPressMsg{Code: 'q', Text: "q"},
		tea.QuitMsg{},
	} {
		m, _ = m.Update(msg)
		if got := len(m.bubbles); got != wantLen {
			t.Errorf("foreign %T mutated stack: len=%d want %d", msg, got, wantLen)
		}
	}
}

// Overlay places the sprite in the configured corner.
func TestOverlayPlacement(t *testing.T) {
	const baseW, baseH = 60, 12

	base := buildBase(baseW, baseH, '.')

	cases := []struct {
		pos          Position
		nameContains string
		checkCorner  func(t *testing.T, lines []string)
	}{
		{
			pos:          BottomRight,
			nameContains: "BottomRight",
			checkCorner: func(t *testing.T, lines []string) {
				// Bottom row should not start with the border in column 0.
				if strings.HasPrefix(lines[baseH-1], "╰") {
					t.Errorf("BottomRight: bubble found at left edge: %q", lines[baseH-1])
				}
				// Top-left corner should be untouched.
				if !strings.HasPrefix(lines[0], "....") {
					t.Errorf("BottomRight: top-left was overwritten: %q", lines[0])
				}
			},
		},
		{
			pos:          BottomLeft,
			nameContains: "BottomLeft",
			checkCorner: func(t *testing.T, lines []string) {
				if !strings.HasPrefix(lines[baseH-1], "╰") {
					t.Errorf("BottomLeft: bubble not at left edge of bottom row: %q", lines[baseH-1])
				}
			},
		},
		{
			pos:          TopRight,
			nameContains: "TopRight",
			checkCorner: func(t *testing.T, lines []string) {
				if strings.HasPrefix(lines[0], "╭") {
					t.Errorf("TopRight: bubble found at left edge of top row: %q", lines[0])
				}
				if !strings.HasSuffix(strings.TrimRight(lines[0], " "), "╮") {
					t.Errorf("TopRight: bubble not at right edge of top row: %q", lines[0])
				}
			},
		},
		{
			pos:          TopLeft,
			nameContains: "TopLeft",
			checkCorner: func(t *testing.T, lines []string) {
				if !strings.HasPrefix(lines[0], "╭") {
					t.Errorf("TopLeft: bubble not at left edge of top row: %q", lines[0])
				}
			},
		},
	}
	for _, c := range cases {
		t.Run(c.nameContains, func(t *testing.T) {
			m := New(WithPosition(c.pos))
			m, _ = m.Push("hello", LevelInfo, time.Second)
			out := m.Overlay(base, baseW, baseH, Insets{})
			stripped := ansi.Strip(out)
			lines := strings.Split(stripped, "\n")
			if len(lines) != baseH {
				t.Fatalf("output has %d lines, want %d", len(lines), baseH)
			}
			for i, l := range lines {
				if visibleWidth(l) != baseW {
					t.Errorf("line %d width=%d, want %d (%q)", i, visibleWidth(l), baseW, l)
				}
			}
			c.checkCorner(t, lines)
			// The message must be present somewhere.
			if !strings.Contains(stripped, "hello") {
				t.Errorf("rendered output missing message: %q", stripped)
			}
		})
	}
}

// A bubble larger than the available area must clip rather than panic, and the result
// keeps the requested base dimensions.
func TestOverlayClipsAtEdge(t *testing.T) {
	const baseW, baseH = 10, 3
	base := buildBase(baseW, baseH, '.')

	m := New()
	m, _ = m.Push("a very long notification that exceeds the area dramatically", LevelInfo, time.Second)
	out := m.Overlay(base, baseW, baseH, Insets{})
	stripped := ansi.Strip(out)
	lines := strings.Split(stripped, "\n")
	if len(lines) != baseH {
		t.Errorf("expected %d lines, got %d", baseH, len(lines))
	}
	for i, l := range lines {
		if visibleWidth(l) != baseW {
			t.Errorf("line %d width=%d, want %d", i, visibleWidth(l), baseW)
		}
	}
}

// Insets push the bubble inward by the requested cells, leaving the reserved edges clear.
func TestOverlayHonoursInsets(t *testing.T) {
	const baseW, baseH = 60, 12
	base := buildBase(baseW, baseH, '.')

	m := New(WithPosition(BottomRight))
	m, _ = m.Push("x", LevelInfo, time.Second)
	out := m.Overlay(base, baseW, baseH, Insets{Right: 2, Bottom: 3})
	stripped := ansi.Strip(out)
	lines := strings.Split(stripped, "\n")

	// Right inset = 2 → rightmost 2 columns of every line must remain "..".
	for i, l := range lines {
		if !strings.HasSuffix(l, "..") {
			t.Errorf("line %d right inset violated: %q", i, l)
		}
	}
	// Bottom inset = 3 → bottom 3 lines must be entirely dots.
	for _, idx := range []int{baseH - 1, baseH - 2, baseH - 3} {
		if got := lines[idx]; got != strings.Repeat(".", baseW) {
			t.Errorf("line %d (bottom inset) was overwritten: %q", idx, got)
		}
	}
}

// When no bubble is active, Overlay returns the input verbatim.
func TestOverlayInactiveIsPassthrough(t *testing.T) {
	m := New()
	base := buildBase(20, 5, '.')
	if got := m.Overlay(base, 20, 5, Insets{}); got != base {
		t.Errorf("Overlay (inactive) mutated content: %q vs %q", got, base)
	}
}

// --- helpers ---------------------------------------------------------------

func buildBase(w, h int, ch rune) string {
	row := strings.Repeat(string(ch), w)
	lines := make([]string, h)
	for i := range lines {
		lines[i] = row
	}
	return strings.Join(lines, "\n")
}

// visibleWidth measures the cell width of a line ignoring ANSI escape sequences.
func visibleWidth(s string) int {
	return lipgloss.Width(s)
}
