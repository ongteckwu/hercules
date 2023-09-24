package similarity_compute

import (
	"hercules/src/code_parser"
	"hercules/src/substring_finder"
)

type SubstringIndexesObject struct {
	StartIndex int
	EndIndex   int
}

type SimilarityResults struct {
	Percentage            float64
	Text1SubstringIndexes SubstringIndexesObject
	Text2SubstringIndexes SubstringIndexesObject
}

func ComputeLevenSimilarity(parsedCodeTextObject1 *code_parser.ParsedCodeTextObject,
	parsedCodeTextObject2 *code_parser.ParsedCodeTextObject) *SimilarityResults {
	parsedText1 := parsedCodeTextObject1.ParsedCodeText
	parsedText2 := parsedCodeTextObject2.ParsedCodeText

	findParsedSSResult1 := substring_finder.FindSubstring(parsedText1, parsedText2)
	findParsedSSResult2 := substring_finder.FindSubstring(parsedText2, parsedText1)

	// get the higher percentage
	var higherPercentage float64
	if findParsedSSResult1.Percentage > findParsedSSResult2.Percentage {
		higherPercentage = findParsedSSResult1.Percentage
	} else {
		higherPercentage = findParsedSSResult2.Percentage
	}

	text1SubstringIndexes := SubstringIndexesObject{
		StartIndex: parsedCodeTextObject1.FindOriginalIndex(findParsedSSResult1.StartIndex),
		EndIndex:   parsedCodeTextObject1.FindOriginalIndex(findParsedSSResult1.EndIndex),
	}

	text2SubstringIndexes := SubstringIndexesObject{
		StartIndex: parsedCodeTextObject2.FindOriginalIndex(findParsedSSResult2.StartIndex),
		EndIndex:   parsedCodeTextObject2.FindOriginalIndex(findParsedSSResult2.EndIndex),
	}

	similarityResults := SimilarityResults{
		Percentage:            higherPercentage,
		Text1SubstringIndexes: text1SubstringIndexes,
		Text2SubstringIndexes: text2SubstringIndexes,
	}
	return &similarityResults
}
