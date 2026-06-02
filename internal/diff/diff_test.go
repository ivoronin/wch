package diff

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestLCS(t *testing.T) {
	eq := func(a, b []string) func(i, j int) float64 {
		return func(i, j int) float64 {
			if a[i] == b[j] {
				return 1
			}
			return 0
		}
	}
	cases := []struct {
		name string
		a, b []string
		want [][2]int
	}{
		{"bothEmpty", nil, nil, [][2]int{}},
		{"oldEmpty", nil, []string{"a", "b"}, [][2]int{}},
		{"newEmpty", []string{"a", "b"}, nil, [][2]int{}},
		{"identity", []string{"x", "y", "z"}, []string{"x", "y", "z"}, [][2]int{{0, 0}, {1, 1}, {2, 2}}},
		{"disjoint", []string{"a", "b"}, []string{"c", "d"}, [][2]int{}},
		{"interior", []string{"x", "M", "y"}, []string{"p", "M", "q"}, [][2]int{{1, 1}}},
		{"prepend", []string{"b", "c"}, []string{"a", "b", "c"}, [][2]int{{0, 1}, {1, 2}}},
	}
	for _, c := range cases {
		got := lcs(len(c.a), len(c.b), eq(c.a, c.b))
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: lcs=%v want %v", c.name, got, c.want)
		}
	}
}

func TestTokenize(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"1/1", []string{"1", "/", "1"}},
		{"a    b", []string{"a", "    ", "b"}},
		{"pod-1 1/1 Running 5m", []string{"pod", "-", "1", " ", "1", "/", "1", " ", "Running", " ", "5m"}},
		{"café", []string{"café"}},
		{"", nil},
	}
	for _, c := range cases {
		if got := tokenize(c.in); !reflect.DeepEqual(got, c.want) {
			t.Errorf("tokenize(%q)=%v want %v", c.in, got, c.want)
		}
	}
}

func TestWordDiff(t *testing.T) {
	cases := []struct {
		name     string
		old, new string
		want     []string // texts of the Changed spans
	}{
		{"wholeWordReplace", "icecream", "beer", []string{"beer"}},
		{"wholeWordReplaceLong", "Running", "Pending", []string{"Pending"}},
		{"subTokenPunct", "1/1", "1/2", []string{"2"}},
		{"ageTick", "pod-1 1/1 Running 5m", "pod-1 1/1 Running 6m", []string{"6m"}},
		{"whitespaceOnly", "a    b", "a  b", nil}, // shifted padding is not a highlightable change
		{"identical", "abc", "abc", nil},
	}
	for _, c := range cases {
		spans := WordDiff(c.old, c.new)
		if got := changedSpans(spans); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: changed spans=%v want %v", c.name, got, c.want)
		}
		// Spans must reconstruct the new line exactly (incl. spacing).
		var b strings.Builder
		for _, s := range spans {
			b.WriteString(s.Text)
		}
		if b.String() != c.new {
			t.Errorf("%s: spans concat=%q want %q", c.name, b.String(), c.new)
		}
	}
}

// The headline bug: character LCS used to keep the shared 'ee' plain. Word-level must mark
// the whole replaced word as one changed span - no plain fragment left behind.
func TestWordDiffNoOrphanChars(t *testing.T) {
	spans := WordDiff("icecream", "beer")
	if len(spans) != 1 || !spans[0].Changed || spans[0].Text != "beer" {
		t.Errorf("got %+v want one changed span \"beer\"", spans)
	}
}

// An inserted duplicate token must mark the inserted (second) instance, not the unchanged
// earlier one. "a b" -> "a a b" tokenizes to [a, " ", a, " ", b]; index 2 is the inserted "a".
func TestTokenDiffDuplicateInsertion(t *testing.T) {
	spans := tokenDiff(tokenize("a b"), tokenize("a a b"))
	if spans[0].Changed {
		t.Errorf("first 'a' (unchanged) must not be Changed")
	}
	if !spans[2].Changed {
		t.Errorf("inserted second 'a' must be Changed")
	}
	if spans[4].Changed {
		t.Errorf("trailing 'b' (unchanged) must not be Changed")
	}
}

// Churn among similar rows: a pod is removed and another added (same age), with one pod
// unchanged. The rows share "1/1 Running 3d" so they are >0.5 similar; the alignment must
// pair the unchanged pod with its exact twin (not a weakly-similar neighbour), so it stays
// plain rather than being word-diffed against an unrelated line.
func TestRenderChurnKeepsUnchangedRowsPlain(t *testing.T) {
	old := text(
		"pod-a 1/1 Running 5m",
		"pod-b 1/1 Running 3d",
	)
	updated := text(
		"pod-b 1/1 Running 3d", // unchanged, shifted up (pod-a removed)
		"pod-c 1/1 Running 3d", // new pod, same age
	)
	lines := Align(old, updated).Lines()
	if lines[0].Kind != LineEqual || lines[0].Text != "pod-b 1/1 Running 3d" {
		t.Errorf("unchanged pod-b must be LineEqual, got kind=%d text=%q", lines[0].Kind, lines[0].Text)
	}
}

// Regression for the coarse false-diff bug: with enough scattered changes that a
// whole-middle DP would exceed alignCellCap, anchoring on the unchanged unique rows must
// keep them plain instead of falling back to positional pairing (which mis-highlighted ~800
// unchanged rows on a real 1900-line `kubectl get pods -A`).
func TestRenderLargeScatteredNoFalseDiffs(t *testing.T) {
	const n = 1600 // n*n > alignCellCap, so a single-segment DP would go coarse
	oldRows := make([]string, n)
	newRows := make([]string, n)
	for i := range oldRows {
		row := fmt.Sprintf("pod-%04d 1/1 Running 9d", i)
		oldRows[i] = row
		newRows[i] = row
	}
	// Scatter changes at top and bottom (forcing a wide middle) plus a mid insertion.
	newRows[0] = "pod-0000 1/1 Running 10d"
	newRows[n-1] = fmt.Sprintf("pod-%04d 1/1 Running 10d", n-1)
	newRows = append(newRows[:800:800], append([]string{"pod-NEWX 0/1 Pending 1s"}, newRows[800:]...)...)

	lines := Align(strings.Join(oldRows, "\n"), strings.Join(newRows, "\n")).Lines()
	kind := make(map[string]LineKind, len(lines))
	for _, ln := range lines {
		kind[ln.Text] = ln.Kind
	}
	for i := 1; i < n-1; i++ { // rows 1..n-2 are unchanged and must stay LineEqual
		row := fmt.Sprintf("pod-%04d 1/1 Running 9d", i)
		if kind[row] != LineEqual {
			t.Fatalf("unchanged row %q kind=%d want LineEqual", row, kind[row])
		}
	}
}

// The primary case: a new row inserts (sorted into place) while neighbouring rows' volatile
// AGE field ticks. The similarity alignment must still pair each shifted row with its old
// self so only the age token highlights, and the new row is fully highlighted.
func TestRenderInsertWithVolatileFields(t *testing.T) {
	old := text(
		"NAME  READY STATUS  AGE",
		"pod-1 1/1   Running 5m",
		"pod-2 1/1   Running 5m",
	)
	updated := text(
		"NAME  READY STATUS  AGE",
		"pod-0 1/1   Running 1s", // new pod, sorts to top
		"pod-1 1/1   Running 6m", // shifted down, AGE ticked
		"pod-2 1/1   Running 6m",
	)
	lines := Align(old, updated).Lines()
	if len(lines) != 4 {
		t.Fatalf("got %d lines want 4", len(lines))
	}
	if lines[0].Kind != LineEqual {
		t.Errorf("header kind=%d want LineEqual", lines[0].Kind)
	}
	if lines[1].Kind != LineAdded || lines[1].Text != "pod-0 1/1   Running 1s" {
		t.Errorf("pod-0 must be LineAdded, got kind=%d text=%q", lines[1].Kind, lines[1].Text)
	}
	for _, i := range []int{2, 3} { // pod-1, pod-2: only the age token changed
		if lines[i].Kind != LineChanged {
			t.Errorf("line %d kind=%d want LineChanged", i, lines[i].Kind)
			continue
		}
		if got := changedSpans(lines[i].Spans); len(got) != 1 || got[0] != "6m" {
			t.Errorf("line %d changed spans=%v want [6m]", i, got)
		}
	}
}
