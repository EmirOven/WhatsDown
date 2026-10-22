package ui

import "postbetasoftware.com/whatsdown/internal/store"

func (c *MainController) appendSentMessageToUI(chatJIDStr string, sentMsg store.StoredMessage) {
	if c.currentChat != nil && c.isSameChat(c.currentChat.JID, chatJIDStr) {
		if _, exists := c.messageRowMap[sentMsg.ID]; !exists {
			var previous *store.StoredMessage
			if len(c.currentMessages) > 0 {
				previous = &c.currentMessages[len(c.currentMessages)-1]
			}
			bubble := c.createMessageRow(sentMsg, previous)
			c.currentMessages = append(c.currentMessages, sentMsg)
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
