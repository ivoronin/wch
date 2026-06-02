package diff

import "strings"

// Span is a run of text within a line together with whether it is a meaningful change worth
// highlighting. Whitespace-only spans are never marked Changed (so shifting column padding
// does not light up), even when they differ.
type Span struct {
	Text    string
	Changed bool
}

// WordDiff breaks newLine into spans against oldLine: a token with no counterpart in oldLine
// (and that is not whitespace-only) is marked Changed. newLine's exact text is preserved, so
// the spans concatenate back to it. A pathologically long line falls back to a single
// changed span.
func WordDiff(oldLine, newLine string) []Span {
	o, n := tokenize(oldLine), tokenize(newLine)
	if len(o)*len(n) > alignCellCap {
		return []Span{{Text: newLine, Changed: true}}
	}
	return tokenDiff(o, n)
}

// tokenDiff returns the new tokens as spans. Byte-identical prefix and suffix tokens are
// peeled before the LCS so that an inserted duplicate marks the inserted instance rather
// than an unchanged earlier one (e.g. "a b" -> "a a b" marks the second "a").
func tokenDiff(old, new []string) []Span {
	n, m := len(old), len(new)
	p := 0
	for p < n && p < m && old[p] == new[p] {
		p++
	}
	s := 0
	for s < n-p && s < m-p && old[n-1-s] == new[m-1-s] {
		s++
	}

	pairs := lcs(n-s-p, m-s-p, func(i, j int) float64 {
		if old[p+i] == new[p+j] {
			return 1
		}
		return 0
	})
	matchedMid := make([]bool, m-s-p)
	for _, pr := range pairs {
		matchedMid[pr[1]] = true
	}

	spans := make([]Span, m)
	for j, t := range new {
		changed := j >= p && j < m-s && !matchedMid[j-p] && strings.TrimSpace(t) != ""
		spans[j] = Span{Text: t, Changed: changed}
	}
	return spans
}
