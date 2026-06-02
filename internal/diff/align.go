// Package diff turns two snapshots of a command's output into a structured description of
// what changed. It is tuned for live "watch"-style output where most lines persist across
// refreshes but their volatile fields (a kubectl AGE, a counter) change every time.
//
// The package is dependency-free (stdlib only) and does no rendering: it reports what
// changed and where lines moved, leaving all styling to the caller.
//
//   - Line alignment (Align): anchor on lines that are byte-identical and unique in both
//     snapshots, then align the gaps between anchors by token similarity (Jaccard), so a row
//     whose AGE ticked still pairs with its old self instead of looking like a delete+insert.
//     This is patience diff with a similarity gap-filler; anchoring keeps it linear on large
//     output.
//   - Structured diff (Lines): each new-output line tagged Equal/Changed/Added, with a
//     word-level Span breakdown for changed lines. The caller styles the Changed spans.
//   - Position mapping (MapLine): where an old line moved to, for preserving a scroll or
//     cursor position across refreshes.
//
// The design is order-preserving: it does not track moves, so reordered output (e.g.
// `kubectl top --sort-by`) shows moved rows as changed.
//
// Internally the concerns form a one-directional chain:
//
//	{lcs, similarity, token} -> align -> { mapline, worddiff } -> diff
package diff

import "strings"

const (
	// simThreshold is the minimum token-overlap (Jaccard) score for two lines to be treated
	// as the same row across a refresh. The identifier token (e.g. a pod NAME) dominates the
	// score, so volatile fields like AGE ticking still match. Tunable.
	simThreshold = 0.5
	// alignCellCap bounds a single gap's similarity DP (and a single line's word DP). A gap
	// with no usable anchors that is larger than this falls back to positional pairing (and
	// MapLine to a similarity scan). With anchoring, this is reached only by mostly-changed
	// output that has no stable lines to anchor on.
	alignCellCap = 2_000_000
)

type opKind uint8

const (
	opMatch opKind = iota
	opInsert
	opDelete
)

type op struct {
	kind   opKind
	oldIdx int // set for opMatch and opDelete, else -1
	newIdx int // set for opMatch and opInsert, else -1
}

// Alignment is the order-preserving correspondence between the lines of two outputs, built
// by anchoring on unchanged unique lines and similarity-aligning the gaps between them. It
// answers where a line moved (MapLine) and what changed per line (Lines).
type Alignment struct {
	ops      []op
	oldLines []string
	newLines []string
	coarse   bool
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

// Align builds a similarity-based, order-preserving alignment of oldText to newText.
func Align(oldText, newText string) Alignment {
	a := Alignment{oldLines: splitLines(oldText), newLines: splitLines(newText)}
	n, m := len(a.oldLines), len(a.newLines)

	// Peel byte-identical common prefix/suffix (cheap; also handles duplicate runs).
	p := 0
	for p < n && p < m && a.oldLines[p] == a.newLines[p] {
		p++
	}
	s := 0
	for s < n-p && s < m-p && a.oldLines[n-1-s] == a.newLines[m-1-s] {
		s++
	}

	for i := range p {
		a.ops = append(a.ops, op{opMatch, i, i})
	}
	a.ops = append(a.ops, a.alignRange(p, n-s, p, m-s)...)
	for k := range s {
		a.ops = append(a.ops, op{opMatch, n - s + k, m - s + k})
	}
	return a
}

// anchor pairs an old-line index with the identical new-line index it pins to.
type anchor struct{ oi, ni int }

// findAnchors returns, increasing on both sides, the lines that are byte-identical and
// appear exactly once in each range. These are unambiguous correspondences: an unchanged row
// pins to its twin no matter how large or scattered the surrounding diff is. The greedy
// increasing scan is optimal for stable-ordered output (insert/delete preserve the relative
// order of survivors); reordered lines simply do not anchor.
func (a *Alignment) findAnchors(oLo, oHi, nLo, nHi int) []anchor {
	oCount := make(map[string]int, oHi-oLo)
	for i := oLo; i < oHi; i++ {
		oCount[a.oldLines[i]]++
	}
	nCount := make(map[string]int, nHi-nLo)
	nFirst := make(map[string]int, nHi-nLo)
	for j := nLo; j < nHi; j++ {
		nl := a.newLines[j]
		nCount[nl]++
		if _, ok := nFirst[nl]; !ok {
			nFirst[nl] = j
		}
	}
	var anchors []anchor
	lastNi := nLo - 1
	for i := oLo; i < oHi; i++ {
		line := a.oldLines[i]
		if oCount[line] == 1 && nCount[line] == 1 {
			if ni := nFirst[line]; ni > lastNi {
				anchors = append(anchors, anchor{i, ni})
				lastNi = ni
			}
		}
	}
	return anchors
}

// alignRange anchors on unchanged unique lines, then similarity-aligns the small gaps
// between them. Anchoring keeps each per-segment alignment tiny even for a huge, scattered
// diff, so the expensive DP never runs over the whole output and unchanged rows are never
// paired positionally with the wrong neighbour.
func (a *Alignment) alignRange(oLo, oHi, nLo, nHi int) []op {
	anchors := a.findAnchors(oLo, oHi, nLo, nHi)
	if len(anchors) == 0 {
		return a.alignGap(oLo, oHi, nLo, nHi)
	}
	var ops []op
	prevO, prevN := oLo, nLo
	for _, an := range anchors {
		ops = append(ops, a.alignGap(prevO, an.oi, prevN, an.ni)...)
		ops = append(ops, op{opMatch, an.oi, an.ni})
		prevO, prevN = an.oi+1, an.ni+1
	}
	ops = append(ops, a.alignGap(prevO, oHi, prevN, nHi)...)
	return ops
}

// alignGap aligns a gap between anchors by token similarity, maximizing total similarity (via
// lcs) so an exact match wins over a look-alike. Within a gap the deletes are emitted before
// the inserts, so a replace block lets Lines pair an old line with its new counterpart for a
// word-level diff. A pathologically large gap (no anchors at all, mostly-changed output)
// falls back to positional pairing, bounded by alignCellCap.
func (a *Alignment) alignGap(oLo, oHi, nLo, nHi int) []op {
	r, c := oHi-oLo, nHi-nLo
	if r == 0 && c == 0 {
		return nil
	}
	if r*c > alignCellCap {
		a.coarse = true
		ops := make([]op, 0, r+c)
		for i := oLo; i < oHi; i++ {
			ops = append(ops, op{opDelete, i, -1})
		}
		for j := nLo; j < nHi; j++ {
			ops = append(ops, op{opInsert, -1, j})
		}
		return ops
	}

	oldSets := make([]map[string]struct{}, r)
	for i := range oldSets {
		oldSets[i] = tokenSet(a.oldLines[oLo+i])
	}
	newSets := make([]map[string]struct{}, c)
	for j := range newSets {
		newSets[j] = tokenSet(a.newLines[nLo+j])
	}
	pairs := lcs(r, c, func(i, j int) float64 {
		if s := jaccard(oldSets[i], newSets[j]); s >= simThreshold {
			return s // weight by similarity so an exact twin (1) beats a look-alike
		}
		return 0
	})

	ops := make([]op, 0, r+c)
	oi, nj := 0, 0
	emitGap := func(oEnd, nEnd int) {
		for ; oi < oEnd; oi++ {
			ops = append(ops, op{opDelete, oLo + oi, -1})
		}
		for ; nj < nEnd; nj++ {
			ops = append(ops, op{opInsert, -1, nLo + nj})
		}
	}
	for _, pr := range pairs {
		emitGap(pr[0], pr[1])
		ops = append(ops, op{opMatch, oLo + pr[0], nLo + pr[1]})
		oi, nj = pr[0]+1, pr[1]+1
	}
	emitGap(r, c)
	return ops
}
