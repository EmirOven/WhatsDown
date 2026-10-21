package markdown

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/diamondburned/gotk4/pkg/glib/v2"
)

func formatLine(line string, stripDelims bool) string {
	trimmed := strings.TrimSpace(line)

	if strings.HasPrefix(trimmed, ">") {
		content := strings.TrimPrefix(trimmed, ">")
		content = strings.TrimPrefix(content, " ")
		escaped := glib.MarkupEscapeText(content)
		if stripDelims {
			return fmt.Sprintf(`<span alpha="70%%">▌ <i>%s</i></span>`, escaped)
		}
		prefix := line[:strings.Index(line, ">")]
		return prefix + `<span alpha="40%">&gt; </span><i>` + escaped + `</i>`
	}

	if strings.HasPrefix(trimmed, "- ") {
		content := strings.TrimPrefix(trimmed, "- ")
		escaped := glib.MarkupEscapeText(content)
		if stripDelims {
			return "• " + escaped
		}
		prefix := line[:strings.Index(line, "- ")]
		return prefix + `<span alpha="40%">- </span>` + escaped
	}

	if len(trimmed) >= 3 && unicode.IsDigit(rune(trimmed[0])) {
		dotIdx := strings.Index(trimmed, ". ")
		if dotIdx > 0 && dotIdx <= 4 {
			numPart := trimmed[:dotIdx+2]
			if _, err := strconv.Atoi(trimmed[:dotIdx]); err == nil {
				content := trimmed[dotIdx+2:]
				escaped := glib.MarkupEscapeText(content)
				if stripDelims {
					return numPart + escaped
				}
				prefix := line[:strings.Index(line, numPart)]
				return prefix + fmt.Sprintf(`<span alpha="50%%">%s</span>`, numPart) + escaped
			}
		}
	}

	return glib.MarkupEscapeText(line)
}
