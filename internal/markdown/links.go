package markdown

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/diamondburned/gotk4/pkg/glib/v2"
)

var urlPattern = regexp.MustCompile("(?i)(?:https?://|www\\.)[^\\s<>\"'`]+")

// Protect raw URLs before formatting: underscores in paths are not italics,
// and URL query strings must never be interpreted as mentions or markup.
func protectURLs(text string) (string, func(string) string) {
	prefix := "\uFFFCurl"
	for strings.Contains(text, prefix) {
		prefix += "x"
	}
	var replacements []string
	var out strings.Builder
	last := 0
	for _, loc := range urlPattern.FindAllStringIndex(text, -1) {
		start, end := loc[0], loc[1]
		if start > 0 {
			prev, _ := utf8.DecodeLastRuneInString(text[:start])
			if unicode.IsLetter(prev) || unicode.IsDigit(prev) || prev == '@' {
				continue
			}
		}
		candidate := text[start:end]
		candidate = strings.TrimRight(candidate, ".,;:!?*~")
		for _, pair := range [][2]string{{"(", ")"}, {"[", "]"}, {"{", "}"}} {
			for strings.HasSuffix(candidate, pair[1]) && strings.Count(candidate, pair[1]) > strings.Count(candidate, pair[0]) {
				candidate = strings.TrimSuffix(candidate, pair[1])
			}
		}
		if start > 0 && text[start-1] == '_' {
			candidate = strings.TrimSuffix(candidate, "_")
		}
		href := candidate
		if strings.HasPrefix(strings.ToLower(href), "www.") {
			href = "https://" + href
		}
		u, err := url.Parse(href)
		if err != nil || u.Hostname() == "" {
			continue
		}
		token := fmt.Sprintf("%s%d\uFFFC", prefix, len(replacements)/2)
		replacements = append(replacements, token, `<a href="`+glib.MarkupEscapeText(href)+`">`+glib.MarkupEscapeText(candidate)+`</a>`)
		out.WriteString(text[last:start])
		out.WriteString(token)
		last = start + len(candidate)
	}
	out.WriteString(text[last:])
	return out.String(), func(formatted string) string {
		if len(replacements) == 0 {
			return formatted
		}
		return strings.NewReplacer(replacements...).Replace(formatted)
	}
}
