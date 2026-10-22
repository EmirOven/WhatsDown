package ui

func (c *MainController) commitSelectedMention() {
	if len(activeMentionState.filteredMembers) == 0 || activeMentionState.selectedIndex < 0 || activeMentionState.selectedIndex >= len(activeMentionState.filteredMembers) {
		c.hideMentionPopup()
		return
	}
	selected := activeMentionState.filteredMembers[activeMentionState.selectedIndex]
	if c.messageEntry == nil {
		c.hideMentionPopup()
		return
	}

	buf := c.messageEntry.Buffer()
	if buf == nil {
		c.hideMentionPopup()
		return
	}

	insertMark := buf.GetInsert()
	cursorIter := buf.IterAtMark(insertMark)
	startIter := buf.StartIter()
	textBefore := buf.Text(startIter, cursorIter, false)

	lastAt := activeMentionState.atStartPos
	if lastAt < 0 || lastAt >= len(textBefore) || textBefore[lastAt] != '@' {
		c.hideMentionPopup()
		return
	}

	atIter := buf.StartIter()
	atIter.ForwardChars(lastAt)

	// Delete from @ to current cursor position
	buf.Delete(atIter, cursorIter)

	tagText := "@" + selected.DisplayName + " "
	activeMentionState.insertedMentions[selected.DisplayName] = selected.JID

	buf.InsertAtCursor(tagText)
	c.hideMentionPopup()
}
