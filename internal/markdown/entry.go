package markdown

import (
	"fmt"
	"strings"

	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/pango"
)

// FormatForEntry parses entry text and builds a *pango.AttrList for GtkEntry.
func FormatForEntry(text string) (*pango.AttrList, error) {
	if text == "" || !HasWhatsAppMarkdown(text) {
		return nil, nil
	}

	segments := splitByCodeBlocks(text)
	var sb strings.Builder

	for _, seg := range segments {
		if seg.isCode {
			escaped := glib.MarkupEscapeText(seg.content)
			sb.WriteString(fmt.Sprintf(`<span alpha="40%%">%s</span><tt>%s</tt><span alpha="40%%">%s</span>`, seg.delims, escaped, seg.delims))
		} else {
			lines := strings.Split(seg.content, "\n")
			for i, line := range lines {
				lines[i] = formatLine(line, false)
			}
			formatted := strings.Join(lines, "\n")
			formatted = formatInline(formatted, false)
			sb.WriteString(formatted)
		}
	}

	formatted := sb.String()
	attrList, plain, _, err := pango.ParseMarkup(formatted, 0)
	if err != nil {
		return nil, err
	}

	if plain != text {
		return nil, fmt.Errorf("length or content mismatch: %q vs %q", plain, text)
	}

	return attrList, nil
}
