// Package overlay owns the cellbuf machinery shared by tui's four renderers (diffrender,
// searchrender, helprender, notify): each one wants to paint styled cells onto a base
// string and read back a styled string. Walk covers the in-place pattern (mutate cells of
// the base buffer); Sprite covers the composition pattern (copy a sprite buffer onto a
// base). Together they collapse the NewBuffer → SetContent → Render → CRLF-strip dance
// to one place, so each renderer keeps only its domain logic (what to highlight, where
// to put the sprite).
package overlay

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/cellbuf"
)

// CellCap bounds the cell grid Walk will build per call. Beyond it, Walk returns the
// base unchanged rather than allocate a huge grid. Empirically large enough for any
// realistic terminal (4M cells ≈ 2000×2000).
const CellCap = 4_000_000

// Walk loads base into a cell buffer of (w, h) and calls mutate to paint cells in
// place. Returns the rendered string with cellbuf's CRLF row separators normalized to
// LF. Returns base unchanged when w == 0 or w*h > CellCap.
func Walk(base string, w, h int, mutate func(*cellbuf.Buffer)) string {
	if w == 0 || w*h > CellCap {
		return base
	}
	buf := cellbuf.NewBuffer(w, h)
	cellbuf.SetContent(buf, base)
	mutate(buf)
	return render(buf)
}

// Sprite composes sprite onto base at (x, y) and returns the result. The result has the
// same dimensions as base (baseW, baseH); cells outside the sprite footprint pass
// through unchanged. cellbuf carries SGR state across the merge so the sprite's styling
// survives intact. The base dimensions are caller-supplied so a renderer can declare
// the canvas size that matters to it (e.g. terminal viewport, not the base string's
// natural width).
func Sprite(base string, baseW, baseH int, sprite string, x, y int) string {
	buf := cellbuf.NewBuffer(baseW, baseH)
	cellbuf.SetContent(buf, base)

	sw, sh := spriteDims(sprite)
	spriteBuf := cellbuf.NewBuffer(sw, sh)
	cellbuf.SetContent(spriteBuf, sprite)

	for dy := range sh {
		for dx := range sw {
			c := spriteBuf.Cell(dx, dy)
			if c == nil || c.Width == 0 {
				continue
			}
			buf.SetCell(x+dx, y+dy, c)
		}
	}
	return render(buf)
}

// MaxDisplayWidth returns the widest display-width line in s, in terminal cells. Used
// by Walk callers that derive the cell-buffer width from the base string itself.
func MaxDisplayWidth(s string) int {
	w := 0
	for _, line := range strings.Split(s, "\n") {
		w = max(w, ansi.StringWidth(line))
	}
	return w
}

// spriteDims returns the natural cell dimensions of a styled sprite string.
func spriteDims(s string) (w, h int) {
	for _, line := range strings.Split(s, "\n") {
		w = max(w, ansi.StringWidth(line))
		h++
	}
	return w, h
}

// render finalises buf to a string with LF row separators (cellbuf emits CRLF).
func render(buf *cellbuf.Buffer) string {
	return strings.ReplaceAll(cellbuf.Render(buf), "\r\n", "\n")
}
