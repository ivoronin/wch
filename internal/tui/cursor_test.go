package tui

import "testing"

func TestCursorConstructors(t *testing.T) {
	if i, ok := noCursor().At(); ok || i != -1 {
		t.Errorf("noCursor().At() = (%d, %v), want (-1, false)", i, ok)
	}
	if i, ok := cursorAtTail(0).At(); ok || i != -1 {
		t.Errorf("cursorAtTail(0).At() = (%d, %v), want (-1, false)", i, ok)
	}
	if i, ok := cursorAtTail(5).At(); !ok || i != 4 {
		t.Errorf("cursorAtTail(5).At() = (%d, %v), want (4, true)", i, ok)
	}
	if i, ok := cursorAt(2).At(); !ok || i != 2 {
		t.Errorf("cursorAt(2).At() = (%d, %v), want (2, true)", i, ok)
	}
}

func TestCursorFollowing(t *testing.T) {
	cases := []struct {
		name string
		c    Cursor
		n    int
		want bool
	}{
		{"empty history with noCursor", noCursor(), 0, true},
		{"empty history with stale cursor", cursorAt(3), 0, true},
		{"at tail of n=5", cursorAt(4), 5, true},
		{"middle of n=5", cursorAt(2), 5, false},
		{"head of n=5", cursorAt(0), 5, false},
		{"noCursor with non-empty", noCursor(), 5, false},
	}
	for _, c := range cases {
		if got := c.c.Following(c.n); got != c.want {
			t.Errorf("%s: Following(%d) = %v, want %v", c.name, c.n, got, c.want)
		}
	}
}

func TestCursorAfterEvict(t *testing.T) {
	cases := []struct {
		in, want int
	}{
		{-1, 0}, // noCursor clamps to 0 after eviction (rare; usually only fires when valid)
		{0, 0},  // the frame they were on was evicted
		{1, 0},
		{4, 3},
	}
	for _, c := range cases {
		got := cursorAt(c.in).AfterEvict()
		if got.Index() != c.want {
			t.Errorf("cursorAt(%d).AfterEvict().Index() = %d, want %d", c.in, got.Index(), c.want)
		}
	}
}

func TestCursorToTail(t *testing.T) {
	if c := noCursor().ToTail(0); c.Valid() {
		t.Errorf("ToTail(0) should be invalid (no frames), got %+v", c)
	}
	if c := cursorAt(0).ToTail(5); c.Index() != 4 {
		t.Errorf("ToTail(5).Index() = %d, want 4", c.Index())
	}
}

func TestCursorMove(t *testing.T) {
	cases := []struct {
		name      string
		from      Cursor
		to, n     int
		wantIdx   int
		wantValid bool
	}{
		{"clamp below", cursorAt(2), -3, 5, 0, true},
		{"clamp above", cursorAt(2), 99, 5, 4, true},
		{"middle", cursorAt(2), 3, 5, 3, true},
		{"empty history", cursorAt(2), 1, 0, -1, false},
		{"from noCursor to valid", noCursor(), 2, 5, 2, true},
	}
	for _, c := range cases {
		got := c.from.Move(c.to, c.n)
		if got.Index() != c.wantIdx || got.Valid() != c.wantValid {
			t.Errorf("%s: Move(%d, %d) = {idx=%d valid=%v}, want {idx=%d valid=%v}",
				c.name, c.to, c.n, got.Index(), got.Valid(), c.wantIdx, c.wantValid)
		}
	}
}
