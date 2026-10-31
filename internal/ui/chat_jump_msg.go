package ui

import (
	"context"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"time"
)

func (c *MainController) jumpToMessage(chatJID, msgID string) {
	if msgID == "" {
		return
	}
	c.autoScrollBottom = false
	c.isAutoScrolling = false
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
		if err == nil && msg == nil {
			msg, _ = c.store.GetMessageByID(context.Background(), msgID)
		}
		if msg != nil {
			c.loadMessagesBefore(chatJID, msg.Timestamp.Add(time.Second))
		}
	}
	c.autoScrollBottom = false
	attempts := 0
	var tryScroll func() bool
	tryScroll = func() bool {
		if c.currentChat == nil || !c.isSameChat(c.currentChat.JID, chatJID) {
			return false
		}
		c.autoScrollBottom = false
		if _, ok := c.messageRowMap[msgID]; ok {
			c.scrollToMessageID(msgID)
			return false
		}
		attempts++
		if attempts < 10 {
			glib.TimeoutAdd(100, tryScroll)
		}
		return false
	}
	glib.TimeoutAdd(100, tryScroll)
}
