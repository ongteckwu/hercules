package code_parser

import (
	"fmt"
	"sort"
	"strings"
)

type LineMetaObject struct {
	LineNumber              int
	LeadingWhitespaceCount  int
	TrailingWhitespaceCount int
}

type ParsedCodeTextObject struct {
	ParsedCodeText string
	LineMeta       []LineMetaObject
	SortedKeys     []int
}

func ParseCodeText(text string) *ParsedCodeTextObject {
	var parsedText strings.Builder // Efficient way to build strings
	var lineMeta []LineMetaObject
	var sortedKeys []int

	lineStart := 0
	lineNumber := 1
	leadingWhitespaceCount := 0
	trailingWhitespaceCount := 0
	inWord := false

	for i, char := range text {
		isWhitespace := char == ' ' || char == '\t'

		if char == '\n' || i == len(text)-1 {
			if i == len(text)-1 && !isWhitespace {
				parsedText.WriteString(text[lineStart : i+1])
			} else {
				parsedText.WriteString(text[lineStart+leadingWhitespaceCount : i-trailingWhitespaceCount])
			}

			// Store metadata
			meta := LineMetaObject{
				LineNumber:              lineNumber,
				LeadingWhitespaceCount:  leadingWhitespaceCount,
				TrailingWhitespaceCount: trailingWhitespaceCount,
			}
			lineMeta = append(lineMeta, meta)
			sortedKeys = append(sortedKeys, lineStart)

			// Reset counters and flags
			lineStart = i + 1
			lineNumber++
			leadingWhitespaceCount = 0
			trailingWhitespaceCount = 0
			inWord = false
			parsedText.WriteString("\n")
			continue
		}

		if isWhitespace {
			if !inWord {
				leadingWhitespaceCount++
			} else {
				trailingWhitespaceCount++
			}
		} else {
			inWord = true
			trailingWhitespaceCount = 0
		}
	}

	parsedCodeTextObject := ParsedCodeTextObject{
		ParsedCodeText: parsedText.String(),
		LineMeta:       lineMeta,
		SortedKeys:     sortedKeys,
	}
	return &parsedCodeTextObject
}

func (parsedCodeTextObject *ParsedCodeTextObject) FindLineStart(index int) (int, int, error) {
	// Find the line that contains index
	sortedKeys := parsedCodeTextObject.SortedKeys
	lineIndex := sort.Search(len(sortedKeys), func(i int) bool { return sortedKeys[i] > index })
	if lineIndex == 0 {
		return -1, -1, fmt.Errorf("no line found")
	}
	// lineIndex is the index of the line that contains index
	// sortedKeys[lineIndex-1] is the startIndex of that line
	return lineIndex, sortedKeys[lineIndex-1], nil
}

func (parsedCodeTextObject *ParsedCodeTextObject) FindOriginalIndex(parsedIndex int) int {
	// Converts the parsedIndex to the original index
	sortedKeys := parsedCodeTextObject.SortedKeys
	lineMeta := parsedCodeTextObject.LineMeta

	lineIndex := sort.Search(len(sortedKeys), func(i int) bool {
		if i == len(sortedKeys)-1 {
			return true
		}
		return sortedKeys[i+1] > parsedIndex
	})

	if lineIndex >= len(lineMeta) || lineIndex < 0 {
		return -1
	}

	// Calculate the difference between parsedIndex and the start of that line in parsedText
	diff := parsedIndex - sortedKeys[lineIndex]

	// Adjust for leading whitespaces to find the original index
	originalIndex := sortedKeys[lineIndex] + lineMeta[lineIndex].LeadingWhitespaceCount + diff

	return originalIndex
}
