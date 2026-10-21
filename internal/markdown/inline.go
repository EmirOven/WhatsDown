package markdown

import (
	"fmt"
	"strings"
	"unicode"
)

func formatInline(text string, stripDelims bool) string {
	delims := []struct {
		char     rune
		openTag  string
		closeTag string
	}{
		{'*', "<b>", "</b>"},
		{'_', "<i>", "</i>"},
		{'~', "<s>", "</s>"},
	}

	for _, d := range delims {
		text = parseDelimitedSpan(text, d.char, d.openTag, d.closeTag, stripDelims)
	}
	return text
}

func parseDelimitedSpan(text string, delim rune, openTag, closeTag string, stripDelims bool) string {
	var sb strings.Builder
	runes := []rune(text)
	n := len(runes)
	i := 0

	for i < n {
		r := runes[i]
		if r == delim && isValidOpenDelim(runes, i) {
			closeIdx := -1
			for j := i + 1; j < n; j++ {
				if runes[j] == delim && isValidCloseDelim(runes, j, i) {
					closeIdx = j
					break
				}
				if runes[j] == '\n' {
					break
				}
				if runes[j] == '<' {
					for j < n && runes[j] != '>' {
						j++
					}
				}
			}

			if closeIdx != -1 && closeIdx > i+1 {
				inner := string(runes[i+1 : closeIdx])
				if stripDelims {
					sb.WriteString(openTag)
					sb.WriteString(inner)
					sb.WriteString(closeTag)
				} else {
					sb.WriteString(fmt.Sprintf(`<span alpha="40%%">%c</span>`, delim))
					sb.WriteString(openTag)
					sb.WriteString(inner)
					sb.WriteString(closeTag)
					sb.WriteString(fmt.Sprintf(`<span alpha="40%%">%c</span>`, delim))
				}
				i = closeIdx + 1
				continue
			}
		}

		sb.WriteRune(r)
		i++
	}
	return sb.String()
}

func isValidOpenDelim(runes []rune, idx int) bool {
	n := len(runes)
	if idx+1 >= n || unicode.IsSpace(runes[idx+1]) {
		return false
	}
	if idx > 0 && (unicode.IsLetter(runes[idx-1]) || unicode.IsDigit(runes[idx-1])) {
		return false
	}
	return true
}

func isValidCloseDelim(runes []rune, idx int, openIdx int) bool {
	n := len(runes)
	if idx-1 < 0 || unicode.IsSpace(runes[idx-1]) {
		return false
	}
	if idx <= openIdx+1 {
		return false
	}
	if idx+1 < n && (unicode.IsLetter(runes[idx+1]) || unicode.IsDigit(runes[idx+1])) {
		return false
	}
	return true
}
