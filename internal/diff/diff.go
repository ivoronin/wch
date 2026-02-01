package diff

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	difflib "github.com/sergi/go-diff/diffmatchpatch"
)

var (
	insertStyle = lipgloss.NewStyle().Background(lipgloss.Color("#2E7D32")) // Green background
)

// Highlighter computes and highlights diffs between outputs
type Highlighter struct {
	dmp *difflib.DiffMatchPatch
}

// New creates a new diff highlighter
func New() *Highlighter {
	return &Highlighter{
		dmp: difflib.New(),
	}
}

// Highlight returns the new text with changed characters highlighted
// Returns the highlighted text and whether there were any changes
func (h *Highlighter) Highlight(oldText, newText string) (string, bool) {
	if oldText == newText {
		return newText, false
	}

	oldLines := strings.Split(oldText, "\n")
	newLines := strings.Split(newText, "\n")

	var result strings.Builder
	hasChanges := false

	for i, newLine := range newLines {
		if i > 0 {
			result.WriteString("\n")
		}

		var oldLine string
		if i < len(oldLines) {
			oldLine = oldLines[i]
		}

		if oldLine == newLine {
			result.WriteString(newLine)
			continue
		}

		hasChanges = true
		highlighted := h.highlightLine(oldLine, newLine)
		result.WriteString(highlighted)
	}

	return result.String(), hasChanges
}

// highlightLine highlights character-level differences in a single line
func (h *Highlighter) highlightLine(oldLine, newLine string) string {
	diffs := h.dmp.DiffMain(oldLine, newLine, false)

	var result strings.Builder
	for _, d := range diffs {
		switch d.Type {
		case difflib.DiffInsert:
			result.WriteString(insertStyle.Render(d.Text))
		case difflib.DiffEqual:
			result.WriteString(d.Text)
			// DiffDelete: we don't render deleted text
		}
	}

	return result.String()
}
