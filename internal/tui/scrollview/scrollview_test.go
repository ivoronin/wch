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
