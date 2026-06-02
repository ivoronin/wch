package diff

// LineKind classifies a new-output line in the diff.
type LineKind uint8

const (
	LineEqual   LineKind = iota // unchanged from the old snapshot
	LineChanged                 // same row, some tokens changed (see Spans)
	LineAdded                   // no counterpart in the old snapshot
)

// Line is the diff of one new-output line. For LineChanged, Spans is the word-level
// breakdown (concatenating back to Text); for LineEqual and LineAdded it is nil. Deleted old
// lines are not represented - the diff describes the new snapshot.
type Line struct {
	Kind     LineKind
	Text     string // the new-output line
	OldIndex int    // matched old-output line index, or -1
	Spans    []Span
}

// Lines returns the structured diff of the new output: one Line per new-output line, in
// order. Callers render it however they like (Equal plain, Added whole-line highlighted,
// Changed with its Changed spans highlighted). Output length equals the number of new lines.
func (a Alignment) Lines() []Line {
	lines := make([]Line, 0, len(a.newLines))
	var pendDel []int // deletes awaiting an insert to pair with (an in-place replace)
	di := 0
	for _, o := range a.ops {
		switch o.kind {
		case opDelete:
			pendDel = append(pendDel, o.oldIdx)
		case opMatch:
			pendDel, di = pendDel[:0], 0 // unpaired deletes are dropped
			nl := a.newLines[o.newIdx]
			if a.oldLines[o.oldIdx] == nl {
				lines = append(lines, Line{Kind: LineEqual, Text: nl, OldIndex: o.oldIdx})
			} else {
				lines = append(lines, Line{Kind: LineChanged, Text: nl, OldIndex: o.oldIdx, Spans: WordDiff(a.oldLines[o.oldIdx], nl)})
			}
		case opInsert:
			nl := a.newLines[o.newIdx]
			if di < len(pendDel) {
				oldIdx := pendDel[di]
				di++
				lines = append(lines, Line{Kind: LineChanged, Text: nl, OldIndex: oldIdx, Spans: WordDiff(a.oldLines[oldIdx], nl)})
			} else {
				lines = append(lines, Line{Kind: LineAdded, Text: nl, OldIndex: -1})
			}
		}
	}
	return lines
}
