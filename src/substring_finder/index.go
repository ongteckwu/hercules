package substring_finder

import "hercules/src/util"

type SubstringResults struct {
	Percentage float64
	StartIndex int
	EndIndex   int
}

func FindSubstring(needle string, haystack string) SubstringResults {
	// FindSubstring finds the substring in a haystack that is most similar to the needle
	minValue, endIndex := ArgminLevenshtein(needle, haystack)

	_, tempIndex := ArgminLevenshtein(util.Reverse(needle), util.Reverse(haystack))
	startIndex := len(haystack) - tempIndex

	substringResults := SubstringResults{
		Percentage: 1 - (float64(minValue) / float64(endIndex-startIndex)),
		StartIndex: startIndex,
		EndIndex:   endIndex,
	}
	return substringResults
}
