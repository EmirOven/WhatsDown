package emoji_test

import (
	"testing"

	"postbetasoftware.com/whatsdown/internal/emoji"
)

func TestIsSingleEmoji(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"standard single emoji", "😀", true},
		{"single emoji heart with VS16", "❤️", true},
		{"single emoji heart without VS16", "❤", true},
		{"single emoji star", "⭐", true},
		{"single emoji coffee", "☕", true},
		{"single emoji thumbs up", "👍", true},
		{"single emoji thumbs up with skin tone", "👍🏽", true},
		{"single emoji waving hand with skin tone", "👋🏿", true},
		{"single emoji family ZWJ", "👨‍👩‍👧‍👦", true},
		{"single emoji tech worker ZWJ", "👩‍💻", true},
		{"single emoji pride flag ZWJ", "🏳️‍🌈", true},
		{"single emoji heart on fire ZWJ", "❤️‍🔥", true},
		{"single emoji juggling woman", "🤹🏽‍♀️", true},
		{"single emoji country flag", "🇳🇱", true},
		{"single emoji keycap", "1️⃣", true},
		{"single emoji keycap hash", "#️⃣", true},
		{"single emoji with leading/trailing whitespace", "   🎉 \n", true},
		{"two emojis together", "😀😀", false},
		{"two emojis with space", "😀 👍", false},
		{"emoji with text after", "😀 hello", false},
		{"emoji with text before", "hello 😀", false},
		{"regular text", "hello", false},
		{"number text", "123", false},
		{"punctuation only", "!", false},
		{"empty string", "", false},
		{"whitespace only", "   \t\n", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := emoji.IsSingleEmoji(tc.input)
			if got != tc.expected {
				t.Errorf("IsSingleEmoji(%q) = %v; want %v", tc.input, got, tc.expected)
			}
		})
	}
}
