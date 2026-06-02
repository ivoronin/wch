package diff

import "strings"

// tokenSet returns the set of whitespace-separated tokens in line. This coarse, set-based
// tokenization is the basis for row identity: two rows are "the same" when their token sets
// overlap enough (jaccard), so an identifier token (e.g. a pod NAME) keeps a row matched to
// its old self even as volatile fields change. It is deliberately coarser than the word
// tokenizer used for highlighting (token.go).
func tokenSet(line string) map[string]struct{} {
	fields := strings.Fields(line)
	set := make(map[string]struct{}, len(fields))
	for _, f := range fields {
		set[f] = struct{}{}
	}
	return set
}

// jaccard is the overlap of two token sets, |A∩B| / |A∪B|.
func jaccard(a, b map[string]struct{}) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 1 // two empty lines are identical
	}
	if len(a) == 0 || len(b) == 0 {
		return 0 // one empty, one not: no overlap
	}
	inter := 0
	for t := range a {
		if _, ok := b[t]; ok {
			inter++
		}
	}
	union := len(a) + len(b) - inter
	return float64(inter) / float64(union)
}
