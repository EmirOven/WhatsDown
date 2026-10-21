package ui

import "postbetasoftware.com/whatsdown/internal/store"

func (c *MainController) appendSentMessageToUI(chatJIDStr string, sentMsg store.StoredMessage) {
	if c.currentChat != nil && c.isSameChat(c.currentChat.JID, chatJIDStr) {
		if _, exists := c.messageRowMap[sentMsg.ID]; !exists {
			sameMinuteAsPrev := false
			if len(c.currentMessages) > 0 {
				prevMsg := c.currentMessages[len(c.currentMessages)-1]
				sameMinuteAsPrev = c.isSameAuthorAndDisplayTime(prevMsg, sentMsg)
			}
			c.currentMessages = append(c.currentMessages, sentMsg)
			bubble := c.createMessageBubble(sentMsg, sameMinuteAsPrev)
			if c.messageRowMap != nil {
				c.messageRowMap[sentMsg.ID] = bubble
			}
			c.messagesList.Append(bubble)
		}
		c.scrollToBottomAnimated()
	}
	c.loadChats()
	c.loadArchivedChats()
}
