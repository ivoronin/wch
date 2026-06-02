// Package diffrender renders a diff (from internal/diff) into styled terminal output: it
// overlays a highlight foreground on the changed cells while preserving the command's own
// ANSI background and attributes, including state carried across line boundaries. It is the
// optional terminal renderer companion to the presentation-free diff package - kept separate
// so diff stays dependency-free.
package diffrender

import (
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/cellbuf"

	"github.com/ivoronin/wch/internal/diff"
	"github.com/ivoronin/wch/internal/tui/overlay"
)

// Render turns lines (the diff of the ANSI-stripped new output, one Line per new line in
// order) into a styled string by overlaying onto styledOutput (the raw, styled new output):
// changed cells get fg as their foreground, the command's background/attributes are
// preserved, and SGR state carried across lines is honoured. styledOutput's rows align with
// lines by index.
func Render(lines []diff.Line, styledOutput string, fg ansi.Color) string {
	if len(lines) == 0 {
		return ""
	}
	return overlay.Walk(styledOutput, overlay.MaxDisplayWidth(styledOutput), len(lines), func(buf *cellbuf.Buffer) {
		for y, ln := range lines {
			highlightRow(buf, y, ln, fg)
		}
	})
}

// highlightRow sets fg on the cells of row y that correspond to changed visible runes of ln.
// Cells are walked left-to-right, each consuming its grapheme's runes (base + combining), to
// stay aligned with the diff's rune-indexed spans.
func highlightRow(buf *cellbuf.Buffer, y int, ln diff.Line, fg ansi.Color) {
	if ln.Kind == diff.LineEqual {
		return // unchanged: keep the command's styling, no highlight
	}

	runes := []rune(ln.Text)
	changed := make([]bool, len(runes))
	switch ln.Kind {
	case diff.LineAdded:
		for i := range changed {
			changed[i] = true
		}
	case diff.LineChanged:
		off := 0
		for _, s := range ln.Spans {
			n := len([]rune(s.Text))
			if s.Changed {
				for i := off; i < off+n && i < len(changed); i++ {
					changed[i] = true
				}
			}
			off += n
		}
	}

	ri := 0
	for x := 0; x < buf.Width() && ri < len(changed); x++ {
		c := buf.Cell(x, y)
		if c == nil || c.Width == 0 {
			continue // padding or the continuation column of a wide rune: not a visible rune
		}
		cnt := 1 + len(c.Comb)
		hot := false
		for i := ri; i < ri+cnt && i < len(changed); i++ {
			if changed[i] {
				hot = true
				break
			}
		}
		if hot {
			c.Style.Fg = fg
		}
		ri += cnt
	}
}
