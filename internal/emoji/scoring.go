package emoji

import "strings"

// ScoreMatch scores how well the query matches an emoji's official name and aliases.
// Higher score indicates a better fit. Returns <= 0 if no match.
func ScoreMatch(query, name string) int {
	q := strings.ToLower(query)
	n := strings.ToLower(name)
	if q == "" {
		return 0
	}
	if n == q {
		return 2000
	}

	words := strings.FieldsFunc(n, func(r rune) bool {
		return r == ' ' || r == ':' || r == '-' || r == '_' || r == ','
	})

	for _, w := range words {
		if w == q {
			return 1500 - len(w)*2
		}
	}

	bestPrefix := 0
	for _, w := range words {
		if strings.HasPrefix(w, q) {
			if s := 1000 + (len(q) * 10) - (len(w) * 2); s > bestPrefix {
				bestPrefix = s
			}
		}
	}
	if bestPrefix > 0 {
		return bestPrefix
	}

	if s := scoreWordFuzzy(words, []rune(q)); s > 0 {
		return s
	}

	for _, w := range words {
		if strings.Contains(w, q) {
			return 500 - len(w)*2
		}
	}

	return scoreFullFuzzy(n, []rune(q))
}
