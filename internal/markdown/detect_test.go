package markdown_test

import (
	"testing"

	"postbetasoftware.com/whatsdown/internal/markdown"
)

func TestHasWhatsAppMarkdownWhileTyping(t *testing.T) {
	cases := []struct {
		text string
		want bool
	}{
		{"", false},
		{"1", false},
		{"1 ", false},
		{"1 a", false},
		{"9 z", false},
		{"123", false},
		{" 1 a ", false},
		{"hello\n1 a", false},
		{"1 é", false},
		{"1.", false},
		{"1. ", false}, // Trailing whitespace is trimmed before detection.
		{"1. a", true},
		{"12. a", true},
		{"hello\n1. a", true},
		{"*bold*", true},
		{"- item", true},
		{"> quote", true},
	}
	for _, tc := range cases {
		t.Run(tc.text, func(t *testing.T) {
			if got := markdown.HasWhatsAppMarkdown(tc.text); got != tc.want {
				t.Fatalf("HasWhatsAppMarkdown(%q) = %v, want %v", tc.text, got, tc.want)
			}
		})
	}
}
