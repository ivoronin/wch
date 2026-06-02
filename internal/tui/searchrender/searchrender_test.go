package searchrender

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// Render applies the SGR 7 reverse attribute to the selected span. After stripping ANSI we
// should see the same visible text; the ANSI sequences should include a SGR 7 enter and a
// SGR 27 leave (or a Reset that turns it off) around the match cells.
func TestRenderAppliesReverseToSelection(t *testing.T) {
	body := "alpha bravo charlie\nfoo bar baz"
	got := Render(body, 0, 6, 5) // "bravo"

	if ansi.Strip(got) != body {
		t.Errorf("stripped output changed:\nbefore: %q\nafter:  %q", body, ansi.Strip(got))
	}
	if !strings.Contains(got, "\x1b[7m") {
		t.Errorf("expected SGR 7 (reverse) in output; got %q", got)
	}
}

// A zero-length selection short-circuits and returns the body untouched.
func TestRenderEmptySelectionPasses(t *testing.T) {
	body := "alpha bravo"
	got := Render(body, 0, 0, 0)
	if got != body {
		t.Errorf("empty selection should pass through; got %q want %q", got, body)
	}
}

// Body wider × taller than overlayCellCap returns body unchanged (degrades gracefully).
func TestRenderTooLargeReturnsBody(t *testing.T) {
	// 2001 cells × 2001 rows ≈ 4M, just over the cap.
	wide := strings.Repeat("x", 2001)
	rows := make([]string, 2001)
	for i := range rows {
		rows[i] = wide
	}
	body := strings.Join(rows, "\n")
	got := Render(body, 0, 0, 5)
	if got != body {
		t.Errorf("oversized body should pass through unchanged")
	}
}
