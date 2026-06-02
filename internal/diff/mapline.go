package diff

// MapLine returns the new-output line index corresponding to old-output line idx, so a
// caller can keep a scroll or cursor position anchored to the same content across a refresh.
// A matched line maps to its counterpart; a deleted line maps to the position where it was;
// an unrelocatable line (coarse fallback, nothing similar) returns idx unchanged.
func (a Alignment) MapLine(idx int) int {
	if idx < 0 {
		return idx
	}
	if a.coarse {
		return a.mapLineByScan(idx)
	}
	curNew := 0
	for _, o := range a.ops {
		switch o.kind {
		case opMatch:
			if o.oldIdx == idx {
				return o.newIdx
			}
			curNew++
		case opInsert:
			curNew++
		case opDelete:
			if o.oldIdx == idx {
				return curNew
			}
		}
	}
	return curNew
}

// mapLineByScan is MapLine's coarse fallback: relocate a single old line by best token
// similarity, biased toward its old index, returning idx unchanged when nothing clears the
// threshold.
func (a Alignment) mapLineByScan(idx int) int {
	if idx >= len(a.oldLines) {
		return min(idx, len(a.newLines))
	}
	target := tokenSet(a.oldLines[idx])
	best := -1
	bestScore := 0.0
	for j, nl := range a.newLines {
		score := jaccard(target, tokenSet(nl))
		if score < simThreshold {
			continue
		}
		if best == -1 || score > bestScore || (score == bestScore && max(j-idx, idx-j) < max(best-idx, idx-best)) {
			best, bestScore = j, score
		}
	}
	if best == -1 {
		return idx
	}
	return best
}
