package ui

import (
	"postbetasoftware.com/whatsdown/internal/emoji"
)

// checkEmojiSuggestion inspects the message buffer for an active colon query.
func (c *MainController) checkEmojiSuggestion() {
	if c.messageEntry == nil {
		return
	}
	buf := c.messageEntry.Buffer()
	if buf == nil {
		return
	}

	insertMark := buf.GetInsert()
	cursorIter := buf.IterAtMark(insertMark)
	startIter := buf.StartIter()
	textBefore := buf.Text(startIter, cursorIter, false)

	offset, query, ok := activeEmojiQuery(textBefore)
	if !ok {
		c.hideEmojiSuggestions()
		return
	}
	c.emojiColonStartPos = offset
	emojis := emoji.Search(query, 8)
	if len(emojis) == 0 {
		c.hideEmojiSuggestions()
		return
	}

	c.showEmojiSuggestions(emojis)
}

// hideEmojiSuggestions conceals the suggestion bar and resets the selection index.
func (c *MainController) hideEmojiSuggestions() {
	if c.emojiSuggestionBar != nil {
		c.emojiSuggestionBar.SetVisible(false)
	}
	c.emojiSuggestions = nil
	c.emojiSuggestionIndex = 0
}
