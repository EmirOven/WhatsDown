package emoji

func scoreWordFuzzy(words []string, qRunes []rune) int {
	best := 0
	for _, w := range words {
		wRunes := []rune(w)
		qi, prevMatch, bonus := 0, -2, 0
		for i, c := range wRunes {
			if qi < len(qRunes) && c == qRunes[qi] {
				if i == 0 {
					bonus += 40
				}
				if i == prevMatch+1 {
					bonus += 25
				}
				prevMatch = i
				qi++
			}
		}
		if qi == len(qRunes) {
			if s := 800 + bonus - (len(w)-len(qRunes))*5; s > best {
				best = s
			}
		}
	}
	return best
}

func scoreFullFuzzy(name string, qRunes []rune) int {
	qi, score, prevMatch := 0, 200, -2
	nRunes := []rune(name)
	for i, c := range nRunes {
		if qi < len(qRunes) && c == qRunes[qi] {
			if i == prevMatch+1 {
				score += 15
			}
			if i == 0 || nRunes[i-1] == ' ' || nRunes[i-1] == ':' || nRunes[i-1] == '_' || nRunes[i-1] == '-' {
				score += 25
			}
			prevMatch = i
			qi++
		}
	}
	if qi == len(qRunes) {
		return score - len(name)
	}
	return -1
}
