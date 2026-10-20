package main

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/pango"
)

// HasWhatsAppMarkdown returns true if the text contains any WhatsApp markdown triggers.
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

type textSegment struct {
	isCode      bool
	isBlockCode bool
	content     string // inner content for code, or raw text for non-code
	delims      string // e.g. "```" or "`"
}

// splitByCodeBlocks splits a string into segments of code blocks and normal text.
func splitByCodeBlocks(text string) []textSegment {
	var segments []textSegment
	n := len(text)
	lastIdx := 0
	i := 0

	for i < n {
		if i+3 <= n && text[i:i+3] == "```" {
			// Find closing ```
			end := strings.Index(text[i+3:], "```")
			if end != -1 {
				closeIdx := i + 3 + end
				if i > lastIdx {
					segments = append(segments, textSegment{content: text[lastIdx:i]})
				}
				segments = append(segments, textSegment{
					isCode:      true,
					isBlockCode: true,
					delims:      "```",
					content:     text[i+3 : closeIdx],
				})
				i = closeIdx + 3
				lastIdx = i
				continue
			}
		} else if text[i] == '`' {
			// Find closing ` on same line
			end := -1
			for j := i + 1; j < n; j++ {
				if text[j] == '\n' {
					break
				}
				if text[j] == '`' {
					end = j
					break
				}
			}
			if end != -1 {
				if i > lastIdx {
					segments = append(segments, textSegment{content: text[lastIdx:i]})
				}
				segments = append(segments, textSegment{
					isCode:      true,
					isBlockCode: false,
					delims:      "`",
					content:     text[i+1 : end],
				})
				i = end + 1
				lastIdx = i
				continue
			}
		}
		i++
	}

	if lastIdx < n {
		segments = append(segments, textSegment{content: text[lastIdx:]})
	}

	return segments
}

// FormatWhatsAppMarkdownForBubble formats message text into Pango markup for chat bubbles and the preview bar.
// It resolves @mentions and renders WhatsApp markdown (bold, italic, strikethrough, monospace, quotes, lists).
func FormatWhatsAppMarkdownForBubble(text string, mentionedJIDs []string, resolveName func(jid string) string) (string, bool) {
	if text == "" {
		return "", false
	}

	segments := splitByCodeBlocks(text)
	var sb strings.Builder

	for _, seg := range segments {
		if seg.isCode {
			escaped := glib.MarkupEscapeText(seg.content)
			if seg.isBlockCode {
				sb.WriteString(fmt.Sprintf(`<span font_family="monospace" background="#1e1e1e" foreground="#dcdcdc"> %s </span>`, escaped))
			} else {
				sb.WriteString(fmt.Sprintf(`<tt>%s</tt>`, escaped))
			}
		} else {
			lines := strings.Split(seg.content, "\n")
			for i, line := range lines {
				lines[i] = formatLineMarkdown(line, true)
			}
			formatted := strings.Join(lines, "\n")
			formatted = formatInlineMarkdown(formatted, true)
			sb.WriteString(formatted)
		}
	}

	result := sb.String()

	// Mentions handling
	hasMentions := false
	if len(mentionedJIDs) > 0 {
		jidMap := make(map[string]string)
		for _, jid := range mentionedJIDs {
			jidMap[jid] = jid
			user := jid
			if idx := strings.Index(user, "@"); idx != -1 {
				user = user[:idx]
			}
			if idx := strings.Index(user, ":"); idx != -1 {
				user = user[:idx]
			}
			if user != "" {
				jidMap[user] = jid
			}
		}

		result = mentionRegex.ReplaceAllStringFunc(result, func(m string) string {
			tag := m[1:]
			targetJID := tag
			if mapped, ok := jidMap[tag]; ok {
				targetJID = mapped
			} else if !strings.Contains(tag, "@") && isAllDigits(tag) && len(tag) >= 5 {
				targetJID = tag + "@s.whatsapp.net"
			}

			label := "@" + tag
			if resolveName != nil {
				resolved := resolveName(targetJID)
				if resolved != "" && resolved != targetJID && !strings.Contains(resolved, "@") {
					label = "@" + resolved
				} else if strings.HasPrefix(targetJID, "+") {
					label = "@" + targetJID
				} else if isAllDigits(tag) && len(tag) >= 5 {
					label = "@+" + tag
				}
			}

			hasMentions = true
			escapedLabel := glib.MarkupEscapeText(label)
			return fmt.Sprintf(`<a href="user:%s"><b>%s</b></a>`, targetJID, escapedLabel)
		})
	}

	return result, hasMentions
}

// FormatWhatsAppMarkdownForEntry parses the entry text and builds a *pango.AttrList
// so that formatting (bold, italic, strikethrough, monospace) is rendered directly inside GtkEntry.
func FormatWhatsAppMarkdownForEntry(text string) (*pango.AttrList, error) {
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
				lines[i] = formatLineMarkdown(line, false)
			}
			formatted := strings.Join(lines, "\n")
			formatted = formatInlineMarkdown(formatted, false)
			sb.WriteString(formatted)
		}
	}

	formatted := sb.String()

	attrList, plain, _, err := pango.ParseMarkup(formatted, 0)
	if err != nil {
		return nil, err
	}

	// Double-check that plain text matches the original text exactly (character-by-character)
	if plain != text {
		return nil, fmt.Errorf("length or content mismatch: %q vs %q", plain, text)
	}

	return attrList, nil
}

// formatLineMarkdown formats quotes ("> ...") and lists ("- ...", "1. ...").
func formatLineMarkdown(line string, stripDelims bool) string {
	trimmed := strings.TrimSpace(line)

	// Quote: > text
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

	// Bullet list: - text or * text
	if strings.HasPrefix(trimmed, "- ") {
		content := strings.TrimPrefix(trimmed, "- ")
		escaped := glib.MarkupEscapeText(content)
		if stripDelims {
			return "• " + escaped
		}
		prefix := line[:strings.Index(line, "- ")]
		return prefix + `<span alpha="40%">- </span>` + escaped
	}

	// Numbered list: e.g. 1. text
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

// formatInlineMarkdown formats *bold*, _italic_, and ~strikethrough~.
func formatInlineMarkdown(text string, stripDelims bool) string {
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

// parseDelimitedSpan parses a specific delimiter character (e.g. '*') and wraps content in tags.
func parseDelimitedSpan(text string, delim rune, openTag, closeTag string, stripDelims bool) string {
	var sb strings.Builder
	runes := []rune(text)
	n := len(runes)
	i := 0

	for i < n {
		r := runes[i]

		// Check if we hit an opening delimiter
		if r == delim && isValidOpenDelim(runes, i) {
			// Search for closing delimiter
			closeIdx := -1
			for j := i + 1; j < n; j++ {
				if runes[j] == delim && isValidCloseDelim(runes, j, i) {
					closeIdx = j
					break
				}
				// Stop on line break if we only format single lines
				if runes[j] == '\n' {
					break
				}
				// Skip existing HTML tags to avoid breaking markup
				if runes[j] == '<' {
					for j < n && runes[j] != '>' {
						j++
					}
				}
			}

			if closeIdx != -1 && closeIdx > i+1 {
				// Valid span found!
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
	// Must have a character after it and it cannot be whitespace
	if idx+1 >= n || unicode.IsSpace(runes[idx+1]) {
		return false
	}
	// Cannot be preceded by an alphanumeric character
	if idx > 0 && (unicode.IsLetter(runes[idx-1]) || unicode.IsDigit(runes[idx-1])) {
		return false
	}
	return true
}

func isValidCloseDelim(runes []rune, idx int, openIdx int) bool {
	n := len(runes)
	// Must not be immediately preceded by whitespace
	if idx-1 < 0 || unicode.IsSpace(runes[idx-1]) {
		return false
	}
	// Must have at least 1 character between open and close
	if idx <= openIdx+1 {
		return false
	}
	// Cannot be followed by an alphanumeric character
	if idx+1 < n && (unicode.IsLetter(runes[idx+1]) || unicode.IsDigit(runes[idx+1])) {
		return false
	}
	return true
}
