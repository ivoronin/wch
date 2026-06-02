package tui

import (
	"fmt"

	"github.com/charmbracelet/x/ansi"

	"github.com/ivoronin/wch/internal/diff"
	"github.com/ivoronin/wch/internal/session"
	"github.com/ivoronin/wch/internal/tui/diffrender"
	"github.com/ivoronin/wch/internal/tui/scrollview"
)

// FrameViewModel owns the rendered viewport: it turns a history Execution into a styled
// body (Frame) and commits a body to the viewport with optional anchor preservation
// (ShowAnchored). The embedded scrollview provides navigation, geometry queries, and
// scrollbar predicates directly on the type, so callers reach for the viewport through
// one seam. The state interface returns "what to display" (state.Body); FrameViewModel
// decides "how to display it".
type FrameViewModel struct {
	scrollview.Scrollview
	session *session.Session
}

// newFrameViewModel constructs the type with a zero-sized viewport; geometry comes from
// the first WindowSizeMsg via SetSize (promoted from the embedded scrollview).
func newFrameViewModel(s *session.Session) FrameViewModel {
	return FrameViewModel{
		Scrollview: scrollview.NewScrollview(0, 0),
		session:    s,
	}
}

// Frame renders the styled body for history index i: command output, optional diff
// highlights against the previous recorded frame (when diffEnabled), and an exit-code
// annotation for non-zero exits. Out-of-range i returns "" so callers can treat it as
// "nothing to display" without a separate predicate.
func (f *FrameViewModel) Frame(i int, diffEnabled bool) string {
	if i < 0 || i >= len(f.session.History) {
		return ""
	}
	exec := f.session.History[i]
	output := exec.Output()
	body := output
	if diffEnabled && i > 0 {
		align := diff.Align(ansi.Strip(f.session.History[i-1].Output()), ansi.Strip(output))
		body = diffrender.Render(align.Lines(), output, insertFg)
	}
	if exec.Error != nil && exec.ExitCode != 0 {
		annot := errorStyle.Render(fmt.Sprintf("Exit code: %d", exec.ExitCode))
		if body == "" {
			body = annot
		} else {
			body += "\n" + annot
		}
	}
	return body
}

// ShowAnchored commits newBody to the viewport while preserving the user's
// eye-on-line invariant relative to prevBody: at the top/bottom edges the
// sticky-edge rule wins (YOffset=0 / GotoBottom); in the middle, diff.Align +
// MapLine translates the old YOffset to its new line after inserts/deletes
// shift content above. Used on every site where the displayed frame changes
// while the cursor (historyIndex) advances or moves: per-exec dispatch and
// per-cursor-move.
//
// Trade-off note: when called as part of a per-exec repaint, this re-derives
// the diff alignment that Frame's diff highlight already computed once for the
// same pair of frames. The dedup gate in Model.dispatchExec keeps duplicate
// ticks from reaching here, so the duplicate only runs on real frame changes
// -- the same frequency the body would have re-aligned at anyway. Resist
// threading a precomputed alignment back through: it poisons the signature
// with a parameter only one caller can supply.
func (f *FrameViewModel) ShowAnchored(newBody, prevBody string) {
	atTop := f.YOffset() == 0
	atBottom := f.AtBottom()

	var newOffset int
	if !atTop && !atBottom {
		anchor := diff.Align(ansi.Strip(prevBody), ansi.Strip(newBody))
		newOffset = anchor.MapLine(f.YOffset())
	}

	f.Scrollview.SetContent(newBody)

	switch {
	case atTop:
		f.SetYOffset(0)
	case atBottom:
		f.GotoBottom()
	default:
		f.SetYOffset(newOffset)
	}
}
