package ui

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// isEmojiSuggestionVisible returns true if the suggestion bar is currently visible.
func (c *MainController) isEmojiSuggestionVisible() bool {
	return c.emojiSuggestionBar != nil && c.emojiSuggestionBar.Visible() && len(c.emojiSuggestions) > 0
}

// navigateEmojiSuggestion shifts the highlighted emoji left or right.
func (c *MainController) navigateEmojiSuggestion(delta int) {
	n := len(c.emojiSuggestions)
	if n == 0 {
		return
	}
	c.emojiSuggestionIndex = (c.emojiSuggestionIndex + delta + n) % n
	c.renderEmojiSuggestionSelection()
}

// renderEmojiSuggestionSelection updates the visual selection state among suggestion buttons.
func (c *MainController) renderEmojiSuggestionSelection() {
	if c.emojiSuggestionBar == nil {
		return
	}
	idx := 0
	for child := c.emojiSuggestionBar.FirstChild(); child != nil; child = gtk.BaseWidget(child).NextSibling() {
		if btn, ok := child.(*gtk.Button); ok {
			if idx == c.emojiSuggestionIndex {
				btn.AddCSSClass("selected")
			} else {
				btn.RemoveCSSClass("selected")
			}
		}
		idx++
	}
}

// commitSelectedEmoji replaces the active :query string with the selected emoji.
func (c *MainController) commitSelectedEmoji() {
	if len(c.emojiSuggestions) == 0 || c.emojiSuggestionIndex < 0 || c.emojiSuggestionIndex >= len(c.emojiSuggestions) {
		c.hideEmojiSuggestions()
		return
	}
	selectedEmoji := c.emojiSuggestions[c.emojiSuggestionIndex]

	if c.messageEntry != nil {
		buf := c.messageEntry.Buffer()
		if buf != nil {
			insertMark := buf.GetInsert()
			cursorIter := buf.IterAtMark(insertMark)
			startIter := buf.StartIter()
			textBefore := buf.Text(startIter, cursorIter, false)

			offset, _, ok := activeEmojiQuery(textBefore)
			if !ok {
				c.hideEmojiSuggestions()
				c.messageEntry.GrabFocus()
				return
			}
			colonIter := buf.IterAtOffset(offset)
			endOffset := cursorIter.Offset()
			closed := strings.HasSuffix(textBefore, ":") && endOffset > offset+1
			if !closed {
				tail := buf.Text(cursorIter, buf.EndIter(), false)
				for _, r := range tail {
					if r == ':' {
						endOffset++
						break
					}
					if !unicode.IsLetter(r) && !unicode.IsDigit(r) && !unicode.IsMark(r) && r != '_' && r != '+' && r != '-' {
						break
					}
					endOffset++
				}
			}
			endIter := buf.IterAtOffset(endOffset)
			buf.BeginUserAction()
			buf.Delete(colonIter, endIter)
			buf.Insert(colonIter, selectedEmoji)
			buf.PlaceCursor(colonIter)
			buf.EndUserAction()
		}
	}

	c.hideEmojiSuggestions()
	if c.messageEntry != nil {
		c.messageEntry.GrabFocus()
	}
}

// GTK iter offsets count Unicode characters rather than UTF-8 bytes.
func activeEmojiQuery(text string) (int, string, bool) {
	colon := strings.LastIndex(text, ":")
	if colon < 0 {
		return 0, "", false
	}
	boundary := func(index int) bool {
		if index == 0 {
			return true
		}
		previous, _ := utf8.DecodeLastRuneInString(text[:index])
		return unicode.IsSpace(previous) || previous == '(' || previous == '['
	}
	valid := func(query string) bool {
		for _, r := range query {
			if unicode.IsSpace(r) || r == ':' {
				return false
			}
		}
		return true
	}
	query := text[colon+1:]
	if colon == len(text)-1 {
		if previous := strings.LastIndex(text[:colon], ":"); previous >= 0 {
			candidate := text[previous+1 : colon]
			if candidate != "" && boundary(previous) && valid(candidate) {
				colon, query = previous, candidate
			}
		}
	}
	if !boundary(colon) || !valid(query) {
		return 0, "", false
	}
	return utf8.RuneCountInString(text[:colon]), query, true
}
