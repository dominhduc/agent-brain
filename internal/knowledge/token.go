package knowledge

func EstimateTokens(text string) int {
	words := 0
	inWord := false
	for _, r := range text {
		if isWordChar(r) {
			if !inWord {
				words++
				inWord = true
			}
		} else {
			inWord = false
		}
	}
	return int(float64(words) * 1.3)
}

func isWordChar(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9') || r == '_' || r == '-'
}
