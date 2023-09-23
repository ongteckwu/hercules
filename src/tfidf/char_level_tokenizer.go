package tfidf

import "unicode"

func TokenizeCharLevelNoAlpha(input string) []string {
	inputRunes := []rune(input)
	tokens := make([]string, 0, len(inputRunes))

	for _, char := range inputRunes {
		if !unicode.IsLetter(char) {
			tokens = append(tokens, string(char))
		}
	}
	return tokens
}
