package scrollview

import (
	"strings"
	"testing"
)

func TestViewNoPanicOnResizeAfterScroll(t *testing.T) {
	// Simulate: wide content, scroll right, then shrink terminal width.
	// This triggers a negative Repeat count in View() because the scrollbar
	// thumb position calculated from the stale XOffset exceeds the new width.
	sv := NewScrollview(80, 10)

	// Content wider than viewport to enable horizontal scrollbar
	wide := strings.Repeat("x", 200)
	lines := make([]string, 20)
	for i := range lines {
		lines[i] = wide
	}
	sv.SetContent(strings.Join(lines, "\n"))

	// Scroll far right
	for range 100 {
		sv.ScrollRight()
	}

	// Shrink viewport (simulates narrowing terminal window)
	sv.SetSize(20, 10)

	// Scroll far right while narrow (large maxXOffset when viewport is small)
	for range 200 {
		sv.ScrollRight()
	}

	// Expand viewport back (simulates widening terminal window).
	// Now XOffset from the narrow state exceeds the valid range for the wider viewport,
	// causing calcScrollbarThumb to produce start+size > Width.
	sv.SetSize(150, 10)

	// View() should not panic
	sv.View()
}

// NeedsVerticalScrollbar / NeedsHorizontalScrollbar reflect whether each bar is currently
// rendered: they require both showBar=true and the content overflowing the corresponding
// axis. Disabling the scrollbar drops both to false.
func TestNeedsScrollbars(t *testing.T) {
	sv := NewScrollview(20, 5)
	sv.SetShowScrollbar(true)

	// Empty / no overflow → neither bar.
	sv.SetContent("hi")
	if sv.NeedsVerticalScrollbar() || sv.NeedsHorizontalScrollbar() {
		t.Errorf("empty content: NeedsV=%v NeedsH=%v, want false/false",
			sv.NeedsVerticalScrollbar(), sv.NeedsHorizontalScrollbar())
	}

	// Tall content → vertical bar; one short line → no horizontal bar.
	tall := strings.Repeat("hi\n", 50)
	sv.SetContent(tall)
	if !sv.NeedsVerticalScrollbar() {
		t.Errorf("tall content: NeedsVerticalScrollbar() = false, want true")
	}
	if sv.NeedsHorizontalScrollbar() {
		t.Errorf("tall (but narrow) content: NeedsHorizontalScrollbar() = true, want false")
	}

	// Wide content → horizontal bar.
	wide := strings.Repeat("x", 200)
	sv.SetContent(wide)
	if !sv.NeedsHorizontalScrollbar() {
		t.Errorf("wide content: NeedsHorizontalScrollbar() = false, want true")
	}

	// Both tall and wide → both bars.
	bothLines := make([]string, 50)
	for i := range bothLines {
		bothLines[i] = wide
	}
	sv.SetContent(strings.Join(bothLines, "\n"))
	if !sv.NeedsVerticalScrollbar() || !sv.NeedsHorizontalScrollbar() {
		t.Errorf("tall+wide content: NeedsV=%v NeedsH=%v, want true/true",
			sv.NeedsVerticalScrollbar(), sv.NeedsHorizontalScrollbar())
	}

	// Disable the scrollbar entirely → both report false even with overflow.
	sv.SetShowScrollbar(false)
	if sv.NeedsVerticalScrollbar() || sv.NeedsHorizontalScrollbar() {
		t.Errorf("scrollbar disabled: NeedsV=%v NeedsH=%v, want false/false",
			sv.NeedsVerticalScrollbar(), sv.NeedsHorizontalScrollbar())
	}
}

// EnsureLineVisible scrolls only when the line is outside the viewport: above → snap to top,
// below → snap to bottom, already inside → no-op.
func TestEnsureLineVisible(t *testing.T) {
	sv := NewScrollview(20, 5)
	sv.SetContent(strings.Repeat("x\n", 50))

	sv.SetYOffset(10)

	// Already visible (within [10, 14]).
	if got := sv.EnsureLineVisible(12); got != 10 {
		t.Errorf("already-visible: YOffset=%d want 10", got)
	}

	// Above viewport → snap line to top.
	if got := sv.EnsureLineVisible(3); got != 3 {
		t.Errorf("above: YOffset=%d want 3", got)
	}

	// Below viewport.
	sv.SetYOffset(10)
	if got := sv.EnsureLineVisible(40); got != 36 { // 40 - 5 + 1
		t.Errorf("below: YOffset=%d want 36", got)
	}
}

// EnsureColumnVisible mirrors EnsureLineVisible on the horizontal axis. When the range is
// wider than the viewport, prefer the left edge to keep the start of the match in view.
func TestEnsureColumnVisible(t *testing.T) {
	sv := NewScrollview(10, 5)
	sv.SetContent(strings.Repeat("x", 100))

	sv.SetXOffset(20)

	// Already visible: col 25 with length 3 fits inside [20, 30).
	if got := sv.EnsureColumnVisible(25, 3); got != 20 {
		t.Errorf("already-visible: XOffset=%d want 20", got)
	}

	// Left of viewport.
	if got := sv.EnsureColumnVisible(5, 3); got != 5 {
		t.Errorf("left: XOffset=%d want 5", got)
	}

	// Right of viewport: range fits → just enough to show it.
	sv.SetXOffset(20)
	if got := sv.EnsureColumnVisible(50, 3); got != 43 { // 50 + 3 - 10
		t.Errorf("right (fits): XOffset=%d want 43", got)
	}

	// Right of viewport, range wider than viewport → keep left edge in view.
	sv.SetXOffset(20)
	if got := sv.EnsureColumnVisible(50, 20); got != 50 {
		t.Errorf("right (oversize): XOffset=%d want 50", got)
	}
}
