package test_util

import (
	"fmt"
	"hercules/src/code_parser"
	"hercules/src/similarity_compute"
	"hercules/src/util"
)

func TestComparison(path1 string, path2 string) {
	data, err := util.MultipleFileRead([]string{path1, path2})
	util.Check(err)
	parsedTextObject1 := code_parser.ParseCodeText(data[path1])
	parsedTextObject2 := code_parser.ParseCodeText(data[path2])

	similarityResult := similarity_compute.ComputeLevenSimilarity(
		parsedTextObject1.ParsedCodeText,
		parsedTextObject2.ParsedCodeText,
	)

	fmt.Println(similarityResult.Percentage)
	text1 := data[path1][similarityResult.Text1SubstringIndexes.StartIndex:similarityResult.Text1SubstringIndexes.EndIndex]
	text2 := data[path2][similarityResult.Text2SubstringIndexes.StartIndex:similarityResult.Text2SubstringIndexes.EndIndex]
	fmt.Println(text1)
	fmt.Println("----------------")
	fmt.Println(text2)
}
