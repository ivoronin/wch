package diff

import (
	"strings"
	"testing"
)

func text(lines ...string) string { return strings.Join(lines, "\n") }

// changedSpans returns the texts of the Changed spans of a line, for concise assertions.
func changedSpans(spans []Span) []string {
	var out []string
	for _, s := range spans {
		if s.Changed {
			out = append(out, s.Text)
		}
	}
	return out
}

func TestSimilarity(t *testing.T) {
	sim := func(a, b string) float64 { return jaccard(tokenSet(a), tokenSet(b)) }
	exact := []struct {
		name string
		a, b string
		want float64
	}{
		{"identical", "a b c", "a b c", 1},
		{"bothEmpty", "", "", 1},
		{"oneEmpty", "a b", "", 0},
		{"disjoint", "pod-1 Running", "svc-9 Pending", 0},
	}
	for _, c := range exact {
		if got := sim(c.a, c.b); got != c.want {
			t.Errorf("%s: similarity=%v want %v", c.name, got, c.want)
		}
	}

	atLeast := []struct {
		name string
		a, b string
	}{
		{"ageTick", "pod-1 1/1 Running 5m", "pod-1 1/1 Running 6m"},
		{"timestampTick", "2024-01-01 12:00:01 starting job", "2024-01-01 12:00:02 starting job"},
	}
	for _, c := range atLeast {
		if got := sim(c.a, c.b); got < simThreshold {
			t.Errorf("%s: similarity=%v want >= %v", c.name, got, simThreshold)
		}
	}
}

func TestMapLineIdentical(t *testing.T) {
	s := text("a", "b", "c")
	a := Align(s, s)
	for i := 0; i < 3; i++ {
		if got := a.MapLine(i); got != i {
			t.Errorf("MapLine(%d)=%d want %d", i, got, i)
		}
	}
}

func TestMapLinePrepend(t *testing.T) {
	a := Align(text("b", "c", "d"), text("a", "b", "c", "d"))
	for old, want := range map[int]int{0: 1, 1: 2, 2: 3} {
		if got := a.MapLine(old); got != want {
			t.Errorf("MapLine(%d)=%d want %d", old, got, want)
		}
	}
}

func TestMapLineAppend(t *testing.T) {
	a := Align(text("a", "b"), text("a", "b", "c"))
	if got := a.MapLine(0); got != 0 {
		t.Errorf("MapLine(0)=%d want 0", got)
	}
	if got := a.MapLine(1); got != 1 {
		t.Errorf("MapLine(1)=%d want 1", got)
	}
}

func TestMapLineKubectlInsertWithAgeTick(t *testing.T) {
	old := text(
		"NAME    READY STATUS  AGE",
		"pod-1   1/1   Running 5m",
		"pod-2   1/1   Running 5m",
		"pod-3   1/1   Running 5m",
	)
	updated := text(
		"NAME    READY STATUS  AGE",
		"pod-0   1/1   Running 1s",
		"pod-1   1/1   Running 6m",
		"pod-2   1/1   Running 6m",
		"pod-3   1/1   Running 6m",
	)
	a := Align(old, updated)
	// Anchored on pod-2 (old idx 2); despite the AGE tick and the inserted pod-0,
	// it relocates to its new home at idx 3.
	if got := a.MapLine(2); got != 3 {
		t.Errorf("MapLine(2)=%d want 3", got)
	}
}

func TestMapLineAnchorDeleted(t *testing.T) {
	a := Align(text("h", "p1", "p2", "p3"), text("h", "p1", "p3"))
	// p2 (old idx 2) removed; anchor lands where it was (new idx 2, now p3) - no teleport.
	if got := a.MapLine(2); got != 2 {
		t.Errorf("MapLine(2)=%d want 2", got)
	}
}

func TestMapLinePastEnd(t *testing.T) {
	a := Align(text("a", "b"), text("a", "b", "c"))
	if got := a.MapLine(100); got != 3 {
		t.Errorf("MapLine(100)=%d want 3", got)
	}
}

func TestMapLineCoarseScan(t *testing.T) {
	a := Alignment{
		coarse:   true,
		oldLines: []string{"alpha 1", "bravo 2", "charlie 3"},
		newLines: []string{"zero 0", "alpha 1", "bravo 2", "charlie 3"},
	}
	if got := a.MapLine(1); got != 2 {
		t.Errorf("coarse MapLine(1)=%d want 2", got)
	}
}

func TestMapLineCoarseNotFound(t *testing.T) {
	a := Alignment{
		coarse:   true,
		oldLines: []string{"unique-xyz abc"},
		newLines: []string{"totally different here"},
	}
	if got := a.MapLine(0); got != 0 {
		t.Errorf("coarse MapLine(0)=%d want 0 (keep offset)", got)
	}
}

func TestLinesPrependAddsOnlyNew(t *testing.T) {
	lines := Align(text("a", "b", "c"), text("x", "a", "b", "c")).Lines()
	want := []LineKind{LineAdded, LineEqual, LineEqual, LineEqual}
	if len(lines) != len(want) {
		t.Fatalf("got %d lines want %d", len(lines), len(want))
	}
	for i, k := range want {
		if lines[i].Kind != k {
			t.Errorf("line %d kind=%d want %d", i, lines[i].Kind, k)
		}
	}
	if lines[0].Text != "x" {
		t.Errorf("added line=%q want %q", lines[0].Text, "x")
	}
}

func TestLinesInPlaceWordChange(t *testing.T) {
	lines := Align("count 5", "count 6").Lines()
	if len(lines) != 1 || lines[0].Kind != LineChanged {
		t.Fatalf("got %+v want one LineChanged", lines)
	}
	if got := changedSpans(lines[0].Spans); len(got) != 1 || got[0] != "6" {
		t.Errorf("changed spans=%v want [6]", got)
	}
}

func TestLinesDeletionDropped(t *testing.T) {
	lines := Align(text("a", "b", "c"), text("a", "c")).Lines()
	if len(lines) != 2 {
		t.Fatalf("got %d lines want 2 (b dropped)", len(lines))
	}
	if lines[0].Text != "a" || lines[1].Text != "c" {
		t.Errorf("texts=%q,%q want a,c", lines[0].Text, lines[1].Text)
	}
	for _, ln := range lines {
		if ln.Kind != LineEqual {
			t.Errorf("line %q kind=%d want LineEqual", ln.Text, ln.Kind)
		}
	}
}

func TestLinesCountEqualsNewLines(t *testing.T) {
	updated := text("a", "X", "Y", "c", "d", "e")
	lines := Align(text("a", "b", "c", "d"), updated).Lines()
	if want := strings.Count(updated, "\n") + 1; len(lines) != want {
		t.Errorf("Lines()=%d want %d", len(lines), want)
	}
}
