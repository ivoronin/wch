package tui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/ivoronin/wch/internal/session"
)

// The status bar must render on a single row. If the left/center/right zones together exceed
// the terminal's contentWidth, the outer statusBarStyle.Width(...).Render wraps the overflow
// onto a second row — the help tail (' quit') visibly disappears off-screen. The center block
// in particular is fragile: its visible width must equal centerBlockWidth exactly, regardless
// of recording state, or the bar tips over the contentWidth budget.
func TestBarSingleLineAtNormalWidths(t *testing.T) {
	for _, w := range []int{60, 80, 100, 120} {
		m := New(Config{Command: "x", Interval: time.Second})
		out, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: 24})
		m = out.(Model)
		bar := m.state.RenderBar(m)
		if got := strings.Count(bar, "\n"); got != 0 {
			t.Errorf("width=%d: bar has %d newlines (want 0); bar wrapped onto a second row", w, got)
		}
	}
}

// renderCenterBlock's visible width must equal centerBlockWidth — the layout math in
// renderBarLayout depends on this. Test both idle and recording states so a future style
// tweak (padding added to recStyle or indicatorStyle) trips the test instead of silently
// stealing cells from the help zone.
func TestCenterBlockWidthMatchesConstant(t *testing.T) {
	m := New(Config{Command: "x", Interval: time.Second})
	out, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = out.(Model)
	if got := lipgloss.Width(m.renderCenterBlock(m.renderIndicator())); got != centerBlockWidth {
		t.Errorf("idle: center block visible width = %d, want %d", got, centerBlockWidth)
	}
}

// TestBarShown documents the single source of truth for whether the bottom bar
// is taking a row of the screen right now. Either the user-configured
// preference (m.prefs.StatusBar) is on, or the active state needs unconditional bar
// feedback (input prompt, picker timeline). searchState is intentionally NOT
// included — under -t we keep its query and counter hidden, consistent with
// the user's "I asked for it off" intent.
func TestBarShown(t *testing.T) {
	tests := []struct {
		name      string
		statusBar bool
		state     state
		want      bool
	}{
		{"view + statusBar on", true, viewState{}, true},
		{"view + statusBar off", false, viewState{}, false},
		{"picker + statusBar off", false, pickerState{}, true},
		{"picker + statusBar on", true, pickerState{}, true},
		{"input on view + statusBar off", false, inputState{prev: viewState{}}, true},
		{"input on view + statusBar on", true, inputState{prev: viewState{}}, true},
		{"input on picker + statusBar off", false, inputState{prev: pickerState{}}, true},
		{"search + statusBar off", false, searchState{prev: viewState{}}, false},
		{"search + statusBar on", true, searchState{prev: viewState{}}, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := New(Config{Command: "x", Interval: time.Second})
			m.prefs.StatusBar = tc.statusBar
			m.state = tc.state
			if got := m.barShown(); got != tc.want {
				t.Errorf("barShown() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestSearchInputBarVisibleWithStatusBarOff: pressing '/' with the status bar
// disabled must reveal the input prompt in place of the bottom viewport row.
// Without the barShown wiring, the bar render gate is m.prefs.StatusBar=false and
// the prompt is hidden — leaving typed characters with no visual feedback.
func TestSearchInputBarVisibleWithStatusBarOff(t *testing.T) {
	m := New(Config{Command: "x", Interval: time.Second})
	m.prefs.StatusBar = false
	m = feed(t, m, tea.WindowSizeMsg{Width: 40, Height: 10})
	m = feed(t, m, execResultMsg{exec: session.Execution{Stdout: podTable("5m", podNames(20))}})

	// Sanity: with statusBar off and viewState, no bar at the bottom.
	pre := m.View().Content
	if got := lipgloss.Height(pre); got != 10 {
		t.Fatalf("pre-press rendered height = %d, want 10", got)
	}
	if got := trailingBarRow(pre); got == m.state.RenderBar(m) {
		t.Fatalf("pre-press: last row already matches m.state.RenderBar(m); test setup is wrong")
	}

	m = pressKey(t, m, '/')

	if _, ok := m.state.(inputState); !ok {
		t.Fatalf("after '/': m.state = %T, want inputState", m.state)
	}
	post := m.View().Content
	if got := lipgloss.Height(post); got != 10 {
		t.Fatalf("post-press rendered height = %d, want 10 (bar must steal a viewport row)", got)
	}
	if got := trailingBarRow(post); got != m.state.RenderBar(m) {
		t.Fatalf("post-press: trailing row does not match m.state.RenderBar(m)\n got: %q\nwant: %q", got, m.state.RenderBar(m))
	}
}

// TestPickerBarVisibleWithStatusBarOff: pressing 'b' (enter picker) with the
// status bar disabled must reveal the picker timeline. On Esc the bar must
// disappear and the viewport must reclaim its row.
func TestPickerBarVisibleWithStatusBarOff(t *testing.T) {
	m := New(Config{Command: "x", Interval: time.Second})
	m.prefs.StatusBar = false
	m = feed(t, m, tea.WindowSizeMsg{Width: 40, Height: 10})
	m = feed(t, m, execResultMsg{exec: session.Execution{Stdout: podTable("5m", podNames(20))}})

	pre := m.View().Content
	if got := trailingBarRow(pre); got == m.state.RenderBar(m) {
		t.Fatalf("pre-press: last row already matches m.state.RenderBar(m); test setup is wrong")
	}

	m = pressKey(t, m, 'b')
	if _, ok := m.state.(pickerState); !ok {
		t.Fatalf("after 'b': m.state = %T, want pickerState", m.state)
	}
	post := m.View().Content
	if got := lipgloss.Height(post); got != 10 {
		t.Fatalf("after 'b': rendered height = %d, want 10", got)
	}
	if got := trailingBarRow(post); got != m.state.RenderBar(m) {
		t.Fatalf("after 'b': trailing row does not match m.state.RenderBar(m)\n got: %q\nwant: %q", got, m.state.RenderBar(m))
	}

	m = feed(t, m, tea.KeyPressMsg{Code: tea.KeyEsc})
	if _, ok := m.state.(viewState); !ok {
		t.Fatalf("after Esc: m.state = %T, want viewState", m.state)
	}
	back := m.View().Content
	if got := trailingBarRow(back); got == m.state.RenderBar(m) {
		t.Fatalf("after Esc: bar still present in trailing row\n got: %q", got)
	}
}

// trailingBarRow returns the last visible row of the rendered View content.
// We compare it directly against m.state.RenderBar(m) to assert that the bar is in fact
// laid out at the bottom of the screen, not just somewhere in the buffer.
func trailingBarRow(content string) string {
	lines := strings.Split(content, "\n")
	return lines[len(lines)-1]
}

// In replay mode, the activity indicator is the ▶ play triangle.
func TestRenderIndicatorReplay(t *testing.T) {
	m := NewReplay(Config{}, preloadedReplaySession(1))
	m = feed(t, m, tea.WindowSizeMsg{Width: 40, Height: 10})
	got := m.renderIndicator()
	if !strings.Contains(got, "▶") {
		t.Errorf("replay indicator should contain ▶; got %q", got)
	}
}

// In live mode, the indicator is one of the live glyphs (no ▶).
func TestRenderIndicatorLiveNotReplay(t *testing.T) {
	m := New(Config{Command: "x", Interval: time.Second})
	m = feed(t, m, tea.WindowSizeMsg{Width: 40, Height: 10})
	got := m.renderIndicator()
	if strings.Contains(got, "▶") {
		t.Errorf("live indicator must not contain ▶; got %q", got)
	}
}
