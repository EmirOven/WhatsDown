package ui

import (
	"strings"

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

			lastColon := strings.LastIndex(textBefore, ":")
			if lastColon != -1 && lastColon <= int(cursorIter.Offset()) {
				colonIter := buf.IterAtOffset(lastColon)
				buf.Delete(colonIter, cursorIter)
				buf.InsertAtCursor(selectedEmoji)
			} else {
				buf.InsertAtCursor(selectedEmoji)
			}
		}
	}

	c.hideEmojiSuggestions()
	if c.messageEntry != nil {
		c.messageEntry.GrabFocus()
	}
}
