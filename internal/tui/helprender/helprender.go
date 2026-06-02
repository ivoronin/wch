// Package helprender composes a centered overlay panel onto pre-rendered terminal content.
// It owns the centering math; the cellbuf composition is delegated to internal/tui/overlay's
// Sprite primitive. The panel itself is built elsewhere (in tui.renderHelpPanel) — this
// package only knows how to paint a string at the center of another string of given
// dimensions, with edge clipping when the panel is larger than the canvas.
package helprender

import (
	"charm.land/lipgloss/v2"

	"github.com/ivoronin/wch/internal/tui/overlay"
)

// Overlay returns content with panel painted on top, centered within (width, height). The
// panel's existing styling (borders, bold spans, etc.) is preserved via cellbuf's nested-SGR
// handling. When the panel exceeds either canvas dimension, its right/bottom is clipped —
// no scrolling, no relayout. Empty panel or non-positive dimensions return content as-is.
func Overlay(content, panel string, width, height int) string {
	if panel == "" || width <= 0 || height <= 0 {
		return content
	}
	x := max(0, (width-lipgloss.Width(panel))/2)
	y := max(0, (height-lipgloss.Height(panel))/2)
	return overlay.Sprite(content, width, height, panel, x, y)
}
