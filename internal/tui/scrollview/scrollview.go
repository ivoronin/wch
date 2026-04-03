package scrollview

import (
	"strings"

	"charm.land/bubbles/v2/viewport"
	"charm.land/lipgloss/v2"
)

// Scrollbar symbols
const (
	vTrackChar = "│"
	vThumbChar = "┃"
	hTrackChar = "─"
	hThumbChar = "━"
)

// Scrollbar styles
var (
	trackStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	thumbStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF"))
)

// Scrollview wraps bubbles viewport with horizontal scrolling and scrollbar rendering.
type Scrollview struct {
	viewport.Model // embedded - navigation and scroll methods auto-promoted

	content     string   // raw content
	lines       []string // cached split lines
	maxWidth    int      // cached max line width
	showBar     bool     // show scrollbar
	totalWidth  int      // user-requested width (content + scrollbar space)
	totalHeight int      // user-requested height (content + scrollbar space)
	needsVBar   bool     // vertical scrollbar needed (computed in updateLayout)
	needsHBar   bool     // horizontal scrollbar needed (computed in updateLayout)
}

// NewScrollview creates a new Scrollview with the given dimensions.
func NewScrollview(width, height int) Scrollview {
	v := Scrollview{
		Model:       viewport.New(viewport.WithWidth(width), viewport.WithHeight(height)),
		showBar:     true,
		totalWidth:  width,
		totalHeight: height,
	}
	return v
}

// SetContent sets the viewport content and preserves scroll position.
func (v *Scrollview) SetContent(content string) {
	v.content = content

	// Cache split lines and max width
	if content == "" {
		v.lines = nil
		v.maxWidth = 0
	} else {
		v.lines = strings.Split(content, "\n")
		v.maxWidth = 0
		for _, line := range v.lines {
			if w := lipgloss.Width(line); w > v.maxWidth {
				v.maxWidth = w
			}
		}
	}

	// Adjust height for horizontal scrollbar BEFORE setting content on Model
	v.updateLayout()

	yoff := v.YOffset()
	xoff := v.XOffset()
	v.Model.SetContent(content)

	// Preserve vertical scroll position (clamped)
	ymax := max(0, v.TotalLineCount()-v.Height())
	v.SetYOffset(min(yoff, ymax))

	// Preserve horizontal scroll (clamped)
	v.SetXOffset(min(xoff, v.maxXOffset()))
}

// updateLayout adjusts embedded viewport dimensions based on scrollbar needs.
func (v *Scrollview) updateLayout() {
	// Compute scrollbar needs using cached values
	v.needsVBar = v.showBar && len(v.lines) > v.totalHeight
	v.needsHBar = v.showBar && v.maxWidth > v.totalWidth

	// Reserve space for scrollbars
	w := v.totalWidth
	h := v.totalHeight
	if v.needsVBar {
		w-- // reserve 1 column for v-scrollbar
	}
	if v.needsHBar {
		h-- // reserve 1 line for h-scrollbar
	}
	v.SetWidth(w)
	v.SetHeight(h)
}

// calcScrollbarThumb computes the start position and size of a scrollbar thumb.
// offset is the current scroll position, visible is the viewport size, total is the content size.
func calcScrollbarThumb(offset, visible, total int) (start, size int) {
	size = max(1, visible*visible/total)
	start = offset * (visible - size) / max(1, total-visible)
	// Clamp so that start+size never exceeds visible (defensive against stale offset)
	start = min(start, max(0, visible-size))
	return
}

// View renders the viewport content with scrollbars.
func (v Scrollview) View() string {
	content := v.Model.View()

	if !v.showBar {
		return content
	}

	lines := strings.Split(content, "\n")
	totalLines := v.TotalLineCount()
	visibleLines := len(lines)

	// Add vertical scrollbar to each line (must append char-by-char to existing lines)
	if v.needsVBar {
		vThumbStart, vThumbSize := calcScrollbarThumb(v.YOffset(), visibleLines, totalLines)

		vTrack := trackStyle.Render(vTrackChar)
		vThumb := thumbStyle.Render(vThumbChar)

		for i := range lines {
			if i >= vThumbStart && i < vThumbStart+vThumbSize {
				lines[i] += vThumb
			} else {
				lines[i] += vTrack
			}
		}
	}

	// Add horizontal scrollbar at bottom (can batch-style whole segments)
	if v.needsHBar {
		hThumbStart, hThumbSize := calcScrollbarThumb(v.XOffset(), v.Width(), v.maxWidth)

		hTrackBefore := strings.Repeat(hTrackChar, hThumbStart)
		hThumb := strings.Repeat(hThumbChar, hThumbSize)
		hTrackAfter := strings.Repeat(hTrackChar, v.Width()-hThumbStart-hThumbSize)

		hBar := lipgloss.JoinHorizontal(lipgloss.Top,
			trackStyle.Render(hTrackBefore),
			thumbStyle.Render(hThumb),
			trackStyle.Render(hTrackAfter),
		)

		// When both scrollbars present, leave corner empty; otherwise hbar takes full width
		if v.needsVBar {
			hBar += " "
		}

		lines = append(lines, hBar)
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// maxXOffset returns the maximum horizontal scroll offset.
func (v *Scrollview) maxXOffset() int {
	return max(0, v.maxWidth-v.Width())
}

// ScrollLeft scrolls the viewport left by one column.
func (v *Scrollview) ScrollLeft() {
	if v.XOffset() > 0 {
		v.SetXOffset(v.XOffset() - 1)
	}
}

// ScrollRight scrolls the viewport right by one column.
func (v *Scrollview) ScrollRight() {
	if v.XOffset() < v.maxXOffset() {
		v.SetXOffset(v.XOffset() + 1)
	}
}

// ScrollLeftPage scrolls the viewport left by one page width.
func (v *Scrollview) ScrollLeftPage() {
	scroll := min(v.XOffset(), v.Width())
	v.SetXOffset(v.XOffset() - scroll)
}

// ScrollRightPage scrolls the viewport right by one page width.
func (v *Scrollview) ScrollRightPage() {
	xmax := v.maxXOffset()
	newOffset := min(v.XOffset()+v.Width(), xmax)
	v.SetXOffset(newOffset)
}

// GotoLeftEdge scrolls to the left edge of content.
func (v *Scrollview) GotoLeftEdge() {
	v.SetXOffset(0)
}

// GotoRightEdge scrolls to the right edge of content.
func (v *Scrollview) GotoRightEdge() {
	v.SetXOffset(v.maxXOffset())
}

// SetSize updates the viewport dimensions.
func (v *Scrollview) SetSize(width, height int) {
	v.totalWidth = width
	v.totalHeight = height
	v.updateLayout()
	// Clamp horizontal scroll to new valid range
	v.SetXOffset(min(v.XOffset(), v.maxXOffset()))
}

// SetShowScrollbar enables or disables the scrollbar.
func (v *Scrollview) SetShowScrollbar(show bool) {
	v.showBar = show
	v.updateLayout()
}
