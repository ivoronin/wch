// Package searchrender draws the in-snapshot search overlay: a single reverse-video span on
// the currently selected match. It is the optional terminal renderer companion to the
// search-matching logic in internal/tui, kept separate so cellbuf-overlay knowledge stays
// local to one package. Mirrors the architecture of internal/tui/diffrender.
package searchrender

import (
	"strings"

	"github.com/charmbracelet/x/cellbuf"

	"github.com/ivoronin/wch/internal/tui/overlay"
)

// Render overlays a reverse-video highlight onto the cells corresponding to the selected
// match (line, col, length in display-width cells). selLine/selCol/selLength describe the
// match in display-width coordinates that line up with cellbuf's grid. The body's existing
// styling (kubectl colors + any diff highlights baked in upstream) is preserved; only the
// reverse attribute (SGR 7) is added on the matched cells, so the terminal swaps the cell's
// own fg/bg without us picking a theme-specific colour. If selLength is 0, body is returned
// unchanged.
func Render(body string, selLine, selCol, selLength int) string {
	if selLength <= 0 {
		return body
	}
	w := overlay.MaxDisplayWidth(body)
	h := strings.Count(body, "\n") + 1
	return overlay.Walk(body, w, h, func(buf *cellbuf.Buffer) {
		for x := selCol; x < selCol+selLength && x < buf.Width(); x++ {
			c := buf.Cell(x, selLine)
			if c == nil {
				continue
			}
			c.Style.Reverse(true)
		}
	})
}
