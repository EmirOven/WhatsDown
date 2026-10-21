package ui

import (
	"context"

	"postbetasoftware.com/whatsdown/internal/store"
)

// UpdateMessageReactions refreshes the reaction pill and bubble for a message.
func (c *MainController) UpdateMessageReactions(chatJID, messageID string) {
	if c.currentChat == nil || !c.isSameChat(c.currentChat.JID, chatJID) {
		return
	}
	c.UpdateSingleMessageRow(messageID)
}

func (c *MainController) updateMessageReactions(chatJID, messageID string) {
	c.UpdateMessageReactions(chatJID, messageID)
}

// UpdateSingleMessageRow refreshes a single message bubble in place.
func (c *MainController) UpdateSingleMessageRow(messageID string) {
	if c.messageRowMap == nil || messageID == "" {
		return
	}
	row, ok := c.messageRowMap[messageID]
	if !ok || row == nil {
		return
	}

	var updatedMsg *store.StoredMessage
	if c.store != nil {
		updatedMsg, _ = c.store.GetMessageByID(context.Background(), messageID)
	}
	if updatedMsg == nil {
		return
	}

	sameMinuteAsPrev := false
	isLastMessage := false
	for i := range c.currentMessages {
		if c.currentMessages[i].ID == messageID {
			if i > 0 {
				sameMinuteAsPrev = c.isSameAuthorAndDisplayTime(c.currentMessages[i-1], *updatedMsg)
			}
			if i == len(c.currentMessages)-1 {
				isLastMessage = true
			}
			c.currentMessages[i] = *updatedMsg
			break
		}
	}

	newRow := c.createMessageBubble(*updatedMsg, sameMinuteAsPrev)
	if sameMinuteAsPrev {
		row.AddCSSClass("message-row-same-minute")
		row.SetMarginTop(2)
	} else {
		row.RemoveCSSClass("message-row-same-minute")
		row.SetMarginTop(8)
	}
	row.SetMarginBottom(0)
	if child := newRow.Child(); child != nil {
		newRow.SetChild(nil)
		row.SetChild(child)
	}

	if isLastMessage {
		c.scrollToBottomAnimated()
	}
}

func (c *MainController) updateSingleMessageRow(messageID string) {
	c.UpdateSingleMessageRow(messageID)
}
