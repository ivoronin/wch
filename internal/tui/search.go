package tui

import (
	"strings"
	"unicode"

	"github.com/charmbracelet/x/ansi"
)

// searchMatch is one occurrence of a query in the searched snapshot, expressed in display-
// width cell coordinates (not byte offsets and not rune indices) so it lines up with the
// cellbuf overlay grid used by searchrender.
type searchMatch struct {
	line   int // 0-based line index in the stripped body
	col    int // 0-based display-width column at which the match starts
	length int // display width of the match
}

// findMatches walks every line of stripped body and collects every occurrence of query,
// using smart-case semantics: case-insensitive when query is all-lowercase, exact when query
// has any uppercase rune. unicode.IsUpper covers non-ASCII alphabets (Ü, Ç, Ñ, …) — limiting
// the check to A-Z would silently fold queries like "Über" or "Lösung" to case-insensitive.
// Returns matches in document order; an empty result means "no match" and the caller must NOT
// enter searchMode (notification is shown instead).
func findMatches(stripped, query string) []searchMatch {
	if query == "" {
		return nil
	}
	haystack := stripped
	needle := query
	if !strings.ContainsFunc(query, unicode.IsUpper) {
		haystack = strings.ToLower(stripped)
		needle = strings.ToLower(query)
	}
	queryWidth := ansi.StringWidth(query)
	var matches []searchMatch
	for lineIdx, line := range strings.Split(haystack, "\n") {
		off := 0      // byte cursor into line
		offWidth := 0 // display width consumed up to off (incremental — saves O(N) per match)
		for {
			i := strings.Index(line[off:], needle)
			if i < 0 {
				break
			}
			byteStart := off + i
			// Add the width of the gap between the previous off and byteStart, not the
			// whole line[:byteStart]. Avoids the O(matches * lineLen) hot path.
			offWidth += ansi.StringWidth(line[off:byteStart])
			matches = append(matches, searchMatch{
				line:   lineIdx,
				col:    offWidth,
				length: queryWidth,
			})
			offWidth += queryWidth
			off = byteStart + len(needle)
			if off >= len(line) {
				break
			}
		}
	}
	return matches
}
