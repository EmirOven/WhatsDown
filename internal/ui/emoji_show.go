package ui

import "github.com/diamondburned/gotk4/pkg/gtk/v4"

// showEmojiSuggestions populates and renders the horizontal emoji recommendation bar.
func (c *MainController) showEmojiSuggestions(emojis []string) {
	if c.emojiSuggestionBar == nil {
		return
	}
	c.emojiSuggestions = emojis
	if c.emojiSuggestionIndex >= len(emojis) || c.emojiSuggestionIndex < 0 {
		c.emojiSuggestionIndex = 0
	}

	for child := c.emojiSuggestionBar.FirstChild(); child != nil; {
		next := gtk.BaseWidget(child).NextSibling()
		c.emojiSuggestionBar.Remove(child)
		child = next
	}

	for i, e := range emojis {
		idx := i
		btn := gtk.NewButton()
		btn.SetLabel(e)
		btn.AddCSSClass("flat")
		btn.AddCSSClass("emoji-suggestion-item")
		if idx == c.emojiSuggestionIndex {
			btn.AddCSSClass("selected")
		}
		btn.ConnectClicked(func() {
			c.emojiSuggestionIndex = idx
			c.commitSelectedEmoji()
		})
		c.emojiSuggestionBar.Append(btn)
	}

	c.emojiSuggestionBar.SetVisible(true)
}
