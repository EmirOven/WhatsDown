package ui

import (
	"fmt"
	"strings"
)

func (c *MainController) performMessageSearch(query string) {
	query = strings.ToLower(strings.TrimSpace(query))
	c.messagesSearchFilter = query
	c.searchMatchIndices = nil
	c.currentSearchMatchIndex = 0

	if query == "" {
		c.updateSearchUI()
		return
	}

	for i, m := range c.currentMessages {
		if strings.Contains(strings.ToLower(m.Text), query) || strings.Contains(strings.ToLower(m.SenderName), query) {
			c.searchMatchIndices = append(c.searchMatchIndices, i)
		}
	}

	if len(c.searchMatchIndices) > 0 {
		c.currentSearchMatchIndex = len(c.searchMatchIndices) - 1
		c.jumpToSearchMatch(c.currentSearchMatchIndex)
	}

	c.updateSearchUI()
}

func (c *MainController) navigateSearchMatch(delta int) {
	if len(c.searchMatchIndices) == 0 {
		return
	}
	n := len(c.searchMatchIndices)
	c.currentSearchMatchIndex = (c.currentSearchMatchIndex + delta + n) % n
	c.jumpToSearchMatch(c.currentSearchMatchIndex)
	c.updateSearchUI()
}

func (c *MainController) updateSearchUI() {
	if c.chatSearchCountLabel == nil {
		return
	}
	if c.messagesSearchFilter == "" {
		c.chatSearchCountLabel.SetText("")
		if c.chatSearchPrevBtn != nil {
			c.chatSearchPrevBtn.SetSensitive(false)
		}
		if c.chatSearchNextBtn != nil {
			c.chatSearchNextBtn.SetSensitive(false)
		}
		return
	}

	if len(c.searchMatchIndices) == 0 {
		c.chatSearchCountLabel.SetText("0 of 0")
		if c.chatSearchPrevBtn != nil {
			c.chatSearchPrevBtn.SetSensitive(false)
		}
		if c.chatSearchNextBtn != nil {
			c.chatSearchNextBtn.SetSensitive(false)
		}
		return
	}

	c.chatSearchCountLabel.SetText(fmt.Sprintf("%d of %d", c.currentSearchMatchIndex+1, len(c.searchMatchIndices)))
	if c.chatSearchPrevBtn != nil {
		c.chatSearchPrevBtn.SetSensitive(true)
	}
	if c.chatSearchNextBtn != nil {
		c.chatSearchNextBtn.SetSensitive(true)
	}
}
