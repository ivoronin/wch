package helprender

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// Empty panel or zero dimensions short-circuit and return content unchanged.
func TestOverlayShortCircuits(t *testing.T) {
	content := "alpha\nbravo\ncharlie\n"
	if got := Overlay(content, "", 80, 24); got != content {
		t.Errorf("empty panel: got != content")
	}
	if got := Overlay(content, "panel", 0, 24); got != content {
		t.Errorf("zero width: got != content")
	}
	if got := Overlay(content, "panel", 80, 0); got != content {
		t.Errorf("zero height: got != content")
	}
}

// A small panel painted onto a larger canvas lands at the center; cells outside the panel
// footprint stay as the original content.
func TestOverlayCentersPanel(t *testing.T) {
	rows := make([]string, 10)
	for i := range rows {
		rows[i] = strings.Repeat("x", 20)
	}
	content := strings.Join(rows, "\n")
	panel := "ABC\nDEF" // 3×2

	got := ansi.Strip(Overlay(content, panel, 20, 10))
	lines := strings.Split(got, "\n")
	if len(lines) < 10 {
		t.Fatalf("output has %d lines, want at least 10", len(lines))
	}
	// Panel is 3×2, canvas 20×10 → top-left at ((20-3)/2, (10-2)/2) = (8, 4).
	if !strings.HasPrefix(lines[4][8:], "ABC") {
		t.Errorf("row 4 col 8: got %q, want prefix \"ABC\"; full row: %q", lines[4][8:], lines[4])
	}
	if !strings.HasPrefix(lines[5][8:], "DEF") {
		t.Errorf("row 5 col 8: got %q, want prefix \"DEF\"; full row: %q", lines[5][8:], lines[5])
	}
	// Row outside the panel keeps the original x-fill.
	if lines[0] != strings.Repeat("x", 20) {
		t.Errorf("row 0: got %q, want unchanged x-fill", lines[0])
	}
}

// A panel larger than the canvas gets clipped at the bottom/right; no panic.
func TestOverlayClipsOversizedPanel(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Overlay panicked with oversized panel: %v", r)
		}
	}()
	content := strings.Repeat("x", 20) + "\n" + strings.Repeat("x", 20)
	// 30×5 panel, canvas 20×2 — way too big in both axes.
	panel := strings.Repeat("A", 30) + "\n" + strings.Repeat("B", 30) + "\n" + strings.Repeat("C", 30) + "\n" + strings.Repeat("D", 30) + "\n" + strings.Repeat("E", 30)
	out := Overlay(content, panel, 20, 2)
	plain := ansi.Strip(out)
	if !strings.Contains(plain, "A") {
		t.Errorf("expected at least some panel content (e.g. 'A') in output; got %q", plain)
	}
}
