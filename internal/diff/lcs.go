package diff

import "slices"

// lcs returns matched index pairs {i, j} in increasing order: an order-preserving set of
// correspondences that maximizes the total weight, where weight(i, j) > 0 means i and j may
// be matched (and how strongly) and weight <= 0 means they may not. The caller decides how
// to treat the gaps between matched pairs (deletes on the old side, inserts on the new).
//
// Maximizing weight rather than match count is what makes an exact counterpart (weight 1)
// win over a merely-similar one (weight < 1): an unchanged row is always paired with its
// identical twin instead of a look-alike neighbour, so it is never spuriously highlighted.
func lcs(r, c int, weight func(i, j int) float64) [][2]int {
	dp := make([][]float64, r+1)
	for i := range dp {
		dp[i] = make([]float64, c+1)
	}
	for i := 1; i <= r; i++ {
		for j := 1; j <= c; j++ {
			best := max(dp[i-1][j], dp[i][j-1])
			if w := weight(i-1, j-1); w > 0 {
				if diag := dp[i-1][j-1] + w; diag > best {
					best = diag
				}
			}
			dp[i][j] = best
		}
	}

	pairs := make([][2]int, 0, min(r, c))
	for i, j := r, c; i > 0 && j > 0; {
		switch w := weight(i-1, j-1); {
		case w > 0 && dp[i][j] == dp[i-1][j-1]+w:
			pairs = append(pairs, [2]int{i - 1, j - 1})
			i--
			j--
		case dp[i-1][j] >= dp[i][j-1]:
			i--
		default:
			j--
		}
	}
	slices.Reverse(pairs)
	return pairs
}
