package markdown

import (
	"strings"
	"unicode"
)

// HasWhatsAppMarkdown returns true if the text contains WhatsApp markdown triggers.
func HasWhatsAppMarkdown(text string) bool {
	if strings.Contains(text, "*") ||
		strings.Contains(text, "_") ||
		strings.Contains(text, "~") ||
		strings.Contains(text, "`") {
		return true
	}
	lines := strings.Split(text, "\n")
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if strings.HasPrefix(trimmed, ">") ||
			strings.HasPrefix(trimmed, "- ") ||
			(len(trimmed) >= 3 && unicode.IsDigit(rune(trimmed[0])) && strings.Contains(trimmed[:4], ". ")) {
			return true
		}
	}
	return false
}
