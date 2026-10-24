package ui

import (
	"context"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"time"
)

func (c *MainController) jumpToMessage(chatJID, msgID string) {
	if c.currentChat == nil || !c.isSameChat(c.currentChat.JID, chatJID) {
		if c.store == nil {
			return
		}
		chat, err := c.store.GetChatOrContact(context.Background(), chatJID)
		if err != nil {
			c.messageActionError(err)
			return
		}
		c.selectChat(chat)
	}
	if c.messageRowMap[msgID] == nil && c.store != nil {
		msg, err := c.store.GetMessage(context.Background(), chatJID, msgID)
		if err != nil || msg == nil {
			return
		}
		c.loadMessagesBefore(chatJID, msg.Timestamp.Add(time.Second))
	}
	// Let layout and the existing chat-load scroll animation finish first.
	glib.TimeoutAdd(250, func() bool {
		if c.currentChat != nil && c.isSameChat(c.currentChat.JID, chatJID) {
			c.scrollToMessageID(msgID)
		}
		return false
	})
}
