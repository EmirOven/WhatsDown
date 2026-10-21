package ui

import (
	"context"
	"log"
	"time"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/store"
)

func (c *MainController) loadMessages(chatJID string) {
	c.messagesList.RemoveAll()
	c.messageRowMap = make(map[string]*gtk.ListBoxRow)
	c.messageStatusLabels = make(map[string]*gtk.Label)

	if c.store == nil {
		return
	}

	msgs, err := c.store.ListMessages(context.Background(), chatJID, time.Time{}, 100)
	if err != nil {
		log.Printf("loading messages for %s: %v", chatJID, err)
		return
	}

	c.currentMessages = msgs
	for i, msg := range msgs {
		sameMinuteAsPrev := false
		if i > 0 {
			sameMinuteAsPrev = c.isSameAuthorAndDisplayTime(msgs[i-1], msg)
		}
		bubble := c.createMessageBubble(msg, sameMinuteAsPrev)
		c.messageRowMap[msg.ID] = bubble
		c.messagesList.Append(bubble)
	}
	c.messagesList.InvalidateFilter()
	c.scrollToBottomAnimated()
}

func (c *MainController) isSameAuthorAndDisplayTime(prev, cur store.StoredMessage) bool {
	isSelfChat := (c.currentChat != nil && c.isSelfJID(c.currentChat.JID)) || c.isSelfJID(prev.ChatJID) || c.isSelfJID(cur.ChatJID)
	if !isSelfChat {
		prevIsOutgoing := prev.FromMe || c.isSelfJID(prev.ChatJID)
		curIsOutgoing := cur.FromMe || c.isSelfJID(cur.ChatJID)
		if prevIsOutgoing != curIsOutgoing {
			return false
		}
		if !prevIsOutgoing && prev.SenderJID != "" && cur.SenderJID != "" && prev.SenderJID != cur.SenderJID {
			return false
		}
	}
	return prev.Timestamp.Local().Format("15:04") == cur.Timestamp.Local().Format("15:04")
}
