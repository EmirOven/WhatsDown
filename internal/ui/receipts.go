package ui

// HandleReceiptUpdate updates the read/delivery ticks on outgoing messages.
func (c *MainController) HandleReceiptUpdate(ids []string, status int) {
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
