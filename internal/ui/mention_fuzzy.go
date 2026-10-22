package ui

import (
	"strings"
)

func normalizeForFuzzy(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		switch r {
		case 'ç':
			b.WriteRune('c')
		case 'ğ':
			b.WriteRune('g')
		case 'ı', 'İ':
			b.WriteRune('i')
		case 'ö':
			b.WriteRune('o')
		case 'ş':
			b.WriteRune('s')
		case 'ü':
			b.WriteRune('u')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func FuzzyMatch(pattern, text string) (bool, int) {
	p := normalizeForFuzzy(pattern)
	t := normalizeForFuzzy(text)
	if p == "" {
		return true, 100
	}
	if idx := strings.Index(t, p); idx != -1 {
		return true, 200 - idx
	}
	pRunes := []rune(p)
	tRunes := []rune(t)
	pIdx := 0
	score := 0
	prevMatch := -1
	for tIdx, r := range tRunes {
		if pIdx < len(pRunes) && r == pRunes[pIdx] {
			score += 10
			if prevMatch != -1 && tIdx == prevMatch+1 {
				score += 20
			}
			prevMatch = tIdx
			pIdx++
		}
	}
	if pIdx == len(pRunes) {
		return true, score
	}
	return false, 0
}
