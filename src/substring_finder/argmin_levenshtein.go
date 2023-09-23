package substring_finder

import (
	"math"
)

// Min3 finds the minimum of three integers
func Min3(a, b, c int) int {
	return int(math.Min(float64(a), math.Min(float64(b), float64(c))))
}

// Calculates the edit distance for substrings in a haystack,
// then finds the index of the needle in the haystack that minimizes the edit distance
func ArgminLevenshtein(needle string, haystack string) (int, int) {
	lenNeedle := len(needle)
	lenHaystack := len(haystack)

	// Create a 2D slice to store the distances
	dp := make([][]int, lenNeedle+1)
	for i := range dp {
		dp[i] = make([]int, lenHaystack+1)
	}

	// Initialize the base cases
	for i := 0; i <= lenNeedle; i++ {
		dp[i][0] = i
	}
	// Fill the first row with zeros as per the modification
	for j := 0; j <= lenHaystack; j++ {
		dp[0][j] = 0
	}

	// Fill the 2D slice with the distances
	for i := 1; i <= lenNeedle; i++ {
		for j := 1; j <= lenHaystack; j++ {
			cost := 0
			if needle[i-1] != haystack[j-1] {
				cost = 1
			}
			dp[i][j] = Min3(
				dp[i-1][j]+1,      // Deletion
				dp[i][j-1]+1,      // Insertion
				dp[i-1][j-1]+cost, // Substitution
			)
		}
	}

	// Find the smallest value in the last row
	minValue := dp[lenNeedle][0]
	endIndex := 0
	for j := 1; j <= lenHaystack; j++ {
		if dp[lenNeedle][j] <= minValue { // we want the last index, so we need the equality
			minValue = dp[lenNeedle][j]
			endIndex = j
		}
	}

	return minValue, endIndex
}
