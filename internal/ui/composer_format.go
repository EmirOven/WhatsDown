package ui

import (
	coreglib "github.com/diamondburned/gotk4/pkg/core/glib"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"strings"
	"unicode"
	"unicode/utf8"
)

func (c *MainController) composerFocused() bool {
	if c.messageEntry == nil {
		return false
	}
	if c.messageEntry.HasFocus() {
		return true
	}
	if c.window == nil || c.window.Focus() == nil {
		return false
	}
	focus := gtk.BaseWidget(c.window.Focus())
	return coreglib.BaseObject(focus).Native() == coreglib.BaseObject(c.messageEntry).Native() || focus.IsAncestor(c.messageEntry)
}

// Selection-only operations use one undo group and keep the transformed text selected.
func (c *MainController) formatComposerSelection(style string) bool {
	if c.messageEntry == nil {
		return false
	}
	buffer := c.messageEntry.Buffer()
	start, end, selected := buffer.SelectionBounds()
	if !selected {
		return false
	}
	text := buffer.Text(start, end, true)
	replacement := text
	switch style {
	case "quote":
		lines := strings.Split(text, "\n")
		quoted := true
		for _, line := range lines {
			if line != "" && !strings.HasPrefix(line, "> ") {
				quoted = false
			}
		}
		for i, line := range lines {
			if quoted {
				lines[i] = strings.TrimPrefix(line, "> ")
			} else if line != "" {
				lines[i] = "> " + line
			}
		}
		replacement = strings.Join(lines, "\n")
		// Keep unselected text outside the quote when the selection starts/ends mid-line.
		if !quoted {
			if start.LineOffset() != 0 {
				replacement = "\n" + replacement
			}
			if !end.EndsLine() && !end.IsEnd() {
				replacement += "\n"
			}
		}
	case "monospace":
		if strings.HasPrefix(text, "```") && strings.HasSuffix(text, "```") && len(text) >= 6 {
			replacement = text[3 : len(text)-3]
		} else {
			replacement = "```" + text + "```"
		}
	default:
		delimiter := map[string]string{"bold": "*", "italic": "_", "strike": "~"}[style]
		if delimiter == "" {
			return false
		}
		lines := strings.Split(text, "\n")
		for i, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				continue
			}
			left := strings.TrimLeftFunc(line, unicode.IsSpace)
			prefix := line[:len(line)-len(left)]
			suffix := left[len(strings.TrimRightFunc(left, unicode.IsSpace)):]
			if len(trimmed) >= 2 && strings.HasPrefix(trimmed, delimiter) && strings.HasSuffix(trimmed, delimiter) {
				trimmed = trimmed[1 : len(trimmed)-1]
			} else {
				trimmed = delimiter + trimmed + delimiter
			}
			lines[i] = prefix + trimmed + suffix
		}
		replacement = strings.Join(lines, "\n")
	}
	offset := start.Offset()
	buffer.BeginUserAction()
	buffer.Delete(start, end)
	buffer.Insert(start, replacement)
	buffer.SelectRange(buffer.IterAtOffset(offset), buffer.IterAtOffset(offset+utf8.RuneCountInString(replacement)))
	buffer.EndUserAction()
	return true
}
