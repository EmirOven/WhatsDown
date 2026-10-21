package emoji

import (
	"sort"
	"strings"
)

var defaultTop = []string{"👍", "❤️", "😂", "🔥", "🎉", "🙏", "🤔", "✨"}

// Search finds up to maxResults emojis matching the query string,
// ranked from best fitting to worst fitting.
func Search(query string, maxResults int) []string {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		if len(defaultTop) > maxResults {
			return defaultTop[:maxResults]
		}
		return defaultTop
	}

	var matches []ScoredEmoji
	seen := make(map[string]bool)

	for _, item := range All() {
		s := ScoreMatch(query, item.Name)
		if s > 0 && !seen[item.Char] {
			seen[item.Char] = true
			if bonus, ok := PopularBonus[item.Char]; ok {
				s += bonus
			}
			matches = append(matches, ScoredEmoji{Char: item.Char, Score: s})
		}
	}

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Score > matches[j].Score
	})

	var results []string
	for i := 0; i < len(matches) && i < maxResults; i++ {
		results = append(results, matches[i].Char)
	}
	return results
}
