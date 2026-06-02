package notify

import (
	"slices"

	"charm.land/lipgloss/v2"

	"github.com/ivoronin/wch/internal/tui/overlay"
)

// Insets describe how much space to leave clear inside the overlay area between the
// bubbles and each edge. Useful to keep bubbles off fixed UI elements (e.g. a status bar at
// the bottom).
type Insets struct{ Top, Right, Bottom, Left int }

// Overlay composes the current bubble stack onto content. content is interpreted as a
// styled string of dimensions width × height (terminal cells); the result is the same
// width × height with bubbles laid into the configured corner, respecting insets. Cells
// outside the bubble footprint show through unchanged.
func (m Model) Overlay(content string, width, height int, in Insets) string {
	if !m.Active() || width <= 0 || height <= 0 {
		return content
	}
	sprite := m.renderSprite()
	sw := lipgloss.Width(sprite)
	sh := lipgloss.Height(sprite)
	x, y := m.corner(width, height, sw, sh, in)
	return overlay.Sprite(content, width, height, sprite, x, y)
}

// renderSprite produces the styled bubble stack as a single string. All bubbles share the
// same width (the widest in the stack) so the column reads cleanly when stacked.
func (m Model) renderSprite() string {
	if len(m.bubbles) == 0 {
		return ""
	}
	// Find the widest message+prefix; pad shorter ones up to that with spaces so each
	// bubble renders at identical visible width.
	contents := make([]string, len(m.bubbles))
	maxContent := 0
	for i, b := range m.bubbles {
		contents[i] = prefixFor(b.level) + b.message
		maxContent = max(maxContent, lipgloss.Width(contents[i]))
	}

	parts := make([]string, len(m.bubbles))
	for i, b := range m.bubbles {
		padded := lipgloss.NewStyle().Width(maxContent).Render(contents[i])
		parts[i] = m.styles[b.level].Render(padded)
	}

	// Newest at the corner: that's the natural append order for Bottom* positions;
	// reverse for Top* so the newest sits at the top.
	if m.position == TopLeft || m.position == TopRight {
		slices.Reverse(parts)
	}
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// corner returns the (x, y) top-left of the sprite for the configured Position, accounting
// for the given insets so the bubble keeps clear of reserved edges.
func (m Model) corner(baseW, baseH, sw, sh int, in Insets) (int, int) {
	switch m.position {
	case BottomRight:
		return baseW - sw - in.Right, baseH - sh - in.Bottom
	case BottomLeft:
		return in.Left, baseH - sh - in.Bottom
	case TopRight:
		return baseW - sw - in.Right, in.Top
	default: // TopLeft
		return in.Left, in.Top
	}
}
