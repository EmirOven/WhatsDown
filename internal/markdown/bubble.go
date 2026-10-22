package markdown

import (
	"fmt"
	"strings"

	"github.com/diamondburned/gotk4/pkg/glib/v2"
)

// FormatForBubble formats message text into Pango markup for chat bubbles.
func FormatForBubble(text string, mentionedJIDs []string, resolveName func(jid string) string) (string, bool) {
	if text == "" {
		return "", false
	}

	segments := splitByCodeBlocks(text)
	var sb strings.Builder
	hasMentions := false

	for _, seg := range segments {
		if seg.isCode {
			escaped := glib.MarkupEscapeText(seg.content)
			if seg.isBlockCode {
				sb.WriteString(fmt.Sprintf(`<span font_family="monospace" background="#1e1e1e" foreground="#dcdcdc"> %s </span>`, escaped))
			} else {
				sb.WriteString(fmt.Sprintf(`<tt>%s</tt>`, escaped))
			}
		} else {
			protected, restoreURLs := protectURLs(seg.content)
			lines := strings.Split(protected, "\n")
			for i, line := range lines {
				lines[i] = formatLine(line, true)
			}
			formatted := strings.Join(lines, "\n")
			formatted = formatInline(formatted, true)
			formatted, mentions := applyMentions(formatted, mentionedJIDs, resolveName)
			hasMentions = hasMentions || mentions
			sb.WriteString(restoreURLs(formatted))
		}
	}

	return sb.String(), hasMentions
}
