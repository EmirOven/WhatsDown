package ui

import (
	"strings"
	"unicode"

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

	lastColon := strings.LastIndex(textBefore, ":")
	if lastColon == -1 {
		c.hideEmojiSuggestions()
		return
	}

	query := textBefore[lastColon+1:]
	for _, r := range query {
		if unicode.IsSpace(r) {
			c.hideEmojiSuggestions()
			return
		}
	}

	if lastColon > 0 {
		prevChar := rune(textBefore[lastColon-1])
		if !unicode.IsSpace(prevChar) && prevChar != '\n' && prevChar != '(' && prevChar != '[' {
			c.hideEmojiSuggestions()
			return
		}
	}

	c.emojiColonStartPos = lastColon
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
