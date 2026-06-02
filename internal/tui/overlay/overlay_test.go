package overlay

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/cellbuf"
)

func TestMaxDisplayWidth(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"", 0},
		{"foo", 3},
		{"foo\nbarbaz", 6},
		{"\x1b[31mred\x1b[0m", 3}, // ANSI sequence ignored
		{"短\n longer line", 12},   // wide rune + longer plain line
		{"short\n\x1b[1mlong with bold\x1b[0m\nmid", 14}, // mixed
	}
	for _, c := range cases {
		if got := MaxDisplayWidth(c.in); got != c.want {
			t.Errorf("MaxDisplayWidth(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestWalkRendersAndMutates(t *testing.T) {
	base := "abc\ndef"
	called := false
	got := Walk(base, 3, 2, func(buf *cellbuf.Buffer) {
		called = true
		buf.Cell(0, 0).Style.Fg = ansi.Red
	})
	if !called {
		t.Errorf("Walk did not invoke mutate")
	}
	if stripped := ansi.Strip(got); stripped != "abc\ndef" {
		t.Errorf("Walk output lost content: stripped=%q want %q", stripped, "abc\ndef")
	}
	if strings.Contains(got, "\r\n") {
		t.Errorf("Walk output contained CRLF: %q", got)
	}
	if !strings.Contains(got, "\x1b[") {
		t.Errorf("Walk did not produce styled output: %q", got)
	}
}

func TestWalkReturnsBaseWhenZeroWidth(t *testing.T) {
	base := "ignored"
	got := Walk(base, 0, 5, func(*cellbuf.Buffer) {
		t.Errorf("mutate must not be called when w == 0")
	})
	if got != base {
		t.Errorf("Walk(w=0) = %q, want base unchanged", got)
	}
}

func TestWalkReturnsBaseWhenCapExceeded(t *testing.T) {
	base := "ignored"
	got := Walk(base, CellCap, 2, func(*cellbuf.Buffer) {
		t.Errorf("mutate must not be called when w*h > CellCap")
	})
	if got != base {
		t.Errorf("Walk(cap exceeded) = %q, want base unchanged", got)
	}
}

func TestSpriteCopiesAtOffset(t *testing.T) {
	base := "abcde\nfghij\nklmno"
	sprite := "XY\nZW"
	got := Sprite(base, 5, 3, sprite, 1, 1)
	stripped := ansi.Strip(got)

	want := "abcde\nfXYij\nkZWno"
	if stripped != want {
		t.Errorf("Sprite composition wrong\n got: %q\nwant: %q", stripped, want)
	}
}

func TestSpritePreservesBaseOutsideFootprint(t *testing.T) {
	base := "aaa\nbbb\nccc"
	sprite := "X"
	got := ansi.Strip(Sprite(base, 3, 3, sprite, 0, 0))
	want := "Xaa\nbbb\nccc"
	if got != want {
		t.Errorf("Sprite leaked outside footprint\n got: %q\nwant: %q", got, want)
	}
}

func TestSpriteUsesLFSeparators(t *testing.T) {
	got := Sprite("ab\ncd", 2, 2, "X", 0, 0)
	if strings.Contains(got, "\r\n") {
		t.Errorf("Sprite output contained CRLF: %q", got)
	}
}
