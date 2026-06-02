package diff

import "unicode"

// tokenize splits s into maximal word-runs, maximal whitespace-runs, and each remaining
// rune as its own token (GitHub's \w+|\s+|[^\w\s], unicode-aware). This fine, ordered
// tokenization drives word-level highlighting (worddiff.go); it is deliberately finer than
// the whitespace token sets used for row identity (similarity.go), which must stay coarse so
// shared punctuation does not blur unrelated rows together.
func tokenize(s string) []string {
	var tokens []string
	runes := []rune(s)
	for i := 0; i < len(runes); {
		switch r := runes[i]; {
		case isWord(r):
			j := i + 1
			for j < len(runes) && isWord(runes[j]) {
				j++
			}
			tokens = append(tokens, string(runes[i:j]))
			i = j
		case unicode.IsSpace(r):
			j := i + 1
			for j < len(runes) && unicode.IsSpace(runes[j]) {
				j++
			}
			tokens = append(tokens, string(runes[i:j]))
			i = j
		default:
			tokens = append(tokens, string(r))
			i++
		}
	}
	return tokens
}

func isWord(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}
