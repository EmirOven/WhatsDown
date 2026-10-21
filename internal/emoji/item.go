package emoji

// Item represents a Unicode emoji and its description.
type Item struct {
	Char string `json:"char"`
	Name string `json:"name"`
}

// ScoredEmoji pairs an emoji character with a relevance match score.
type ScoredEmoji struct {
	Char  string
	Score int
}
