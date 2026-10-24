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
		if c.currentChat != nil {
			updatedMsg, _ = c.store.GetMessage(context.Background(), c.currentChat.JID, messageID)
		}
	}
	if updatedMsg == nil {
		return
	}

	var previous *store.StoredMessage
	isLastMessage := false
	for i := range c.currentMessages {
		if c.currentMessages[i].ID == messageID {
			if i > 0 {
				previous = &c.currentMessages[i-1]
			}
			if i == len(c.currentMessages)-1 {
				isLastMessage = true
			}
			c.currentMessages[i] = *updatedMsg
			break
		}
	}

	newRow := c.createMessageRow(*updatedMsg, previous)
	index := row.Index()
	c.releaseMessageWidget(row)
	c.messagesList.Remove(row)
	c.messagesList.Insert(newRow, index)
	c.messageRowMap[messageID] = newRow

	if isLastMessage {
		c.scrollToBottomAnimated()
	}
}

func (c *MainController) updateSingleMessageRow(messageID string) {
	c.UpdateSingleMessageRow(messageID)
}

// RefreshMessageRows refreshes the currently displayed messages after a batch
// of app-state changes without queuing callbacks for every historical message.
func (c *MainController) RefreshMessageRows() {
	for id := range c.messageRowMap {
		c.UpdateSingleMessageRow(id)
	}
}
