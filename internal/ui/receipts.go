package ui

import "strings"

// HandleReceiptUpdate updates the read/delivery ticks on outgoing messages.
func (c *MainController) HandleReceiptUpdate(ids []string, status int) {
	if c.currentChat != nil && strings.HasSuffix(c.currentChat.JID, "@g.us") {
		c.RefreshReceiptMessages(c.currentChat.JID, ids)
		return
	}
	for _, id := range ids {
		if lbl, ok := c.messageStatusLabels[id]; ok && lbl != nil {
			c.applyStatusToLabel(lbl, status)
		}
		for i := range c.currentMessages {
			if c.currentMessages[i].ID == id {
				if status > c.currentMessages[i].Status {
					c.currentMessages[i].Status = status
				}
				break
			}
		}
	}
}

// insertEmoji inserts the selected emoji into the message entry at the current cursor position.
func (c *MainController) insertEmoji(text string) {
	if c.messageEntry == nil {
		return
	}
	buf := c.messageEntry.Buffer()
	if buf != nil {
		buf.InsertAtCursor(text)
	}
	c.messageEntry.GrabFocus()
	if c.messagePlaceholder != nil {
		c.messagePlaceholder.SetVisible(c.getMessageText() == "")
	}
}

// RefreshReceiptMessages reads the aggregate status rather than applying one
// participant's receipt to the entire group.
func (c *MainController) RefreshReceiptMessages(chat string, ids []string) {
	if c.currentChat == nil || !c.isSameChat(c.currentChat.JID, chat) {
		return
	}
	for _, id := range ids {
		c.UpdateSingleMessageRow(id)
	}
}
