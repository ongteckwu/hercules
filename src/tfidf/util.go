package tfidf

import (
	"hercules/src/util"
	"sort"
)

func GetTopNKeywordsTfIdf(n int, weights map[string]float64) []string {

	// Convert map to slice of Pair
	pairs := make([]util.Pair[float64], 0, len(weights))
	for k, v := range weights {
		pairs = append(pairs, util.Pair[float64]{Key: k, Value: v})
	}

	// Sort the slice based on float64 value
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].Value > pairs[j].Value
	})

	if len(pairs) >= n {
		// get number of elements with same value as the first element
		maxValue := pairs[0].Value
		var numberOfSameValues int
		for i := 1; i < n; i++ {
			if pairs[i].Value == maxValue {
				numberOfSameValues++
			} else {
				break
			}
		}

		if numberOfSameValues > n {
			sortByStringLengthForPairs(pairs)
		} else {
			sortByStringLengthForPairs(pairs[:numberOfSameValues])
		}

		// Keep only top 10 (or fewer if there are fewer than 10 elements)
		if len(pairs) > 10 {
			pairs = pairs[:10]
		}
	} else {
		sortByStringLengthForPairs(pairs)
	}
	keywords := util.Map(pairs, func(pair util.Pair[float64]) string { return pair.Key })
	return keywords
}

func sortByStringLengthForPairs(pairs []util.Pair[float64]) {
	sort.Slice(pairs, func(i, j int) bool {
		return len(pairs[i].Key) > len(pairs[j].Key)
	})
}
