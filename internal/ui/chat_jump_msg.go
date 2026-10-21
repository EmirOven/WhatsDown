package ui

import "github.com/diamondburned/gotk4/pkg/glib/v2"

func (c *MainController) jumpToMessage(chatJID string, msgID string) {
	if c.currentChat == nil || c.currentChat.JID != chatJID {
		for _, ch := range c.chats {
			if ch.JID == chatJID {
				c.selectChat(ch)
				break
			}
		}
		for _, ch := range c.archivedChats {
			if ch.JID == chatJID {
				c.selectChat(ch)
				break
			}
		}
	}

	foundIdx := -1
	for i, m := range c.currentMessages {
		if m.ID == msgID {
			foundIdx = i
			break
		}
	}

	if foundIdx >= 0 {
		row := c.messagesList.RowAtIndex(foundIdx)
		if row != nil {
			row.GrabFocus()
			row.AddCSSClass("message-highlighted")
			glib.TimeoutAdd(2000, func() bool {
				row.RemoveCSSClass("message-highlighted")
				return false
			})
		}
	}
}
