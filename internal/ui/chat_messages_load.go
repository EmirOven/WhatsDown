package ui

import (
	"context"
	"log"
	"time"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/store"
)

func (c *MainController) loadMessages(chatJID string) { c.loadMessagesBefore(chatJID, time.Time{}) }

func (c *MainController) loadMessagesBefore(chatJID string, before time.Time) {
	if c.cancelMessagePreviews != nil {
		c.cancelMessagePreviews()
	}
	c.messagePreviewContext, c.cancelMessagePreviews = context.WithCancel(context.Background())
	if c.messageSelection != nil {
		c.messageSelection.checks = make(map[string]*gtk.CheckButton)
		if c.messageSelection.chat != chatJID {
			c.cancelMessageSelection()
		}
	}
	c.currentMessages = nil
	for child := c.messagesList.FirstChild(); child != nil; child = gtk.BaseWidget(child).NextSibling() {
		c.releaseMessageWidget(child)
	}
	c.messagesList.RemoveAll()
	c.messageRowMap = make(map[string]*gtk.ListBoxRow)
	c.messageStatusLabels = make(map[string]*gtk.Label)

	if c.store == nil {
		return
	}

	msgs, err := c.store.ListMessages(context.Background(), chatJID, before, 100)
	if err != nil {
		log.Printf("loading messages for %s: %v", chatJID, err)
		return
	}

	c.currentMessages = msgs
	for i, msg := range msgs {
		var previous *store.StoredMessage
		if i > 0 {
			previous = &msgs[i-1]
		}
		bubble := c.createMessageRow(msg, previous)
		c.messageRowMap[msg.ID] = bubble
		c.messagesList.Append(bubble)
	}
	c.pruneMessageSelection()
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
	return prev.Timestamp.Local().Format("2006-01-02 15:04") == cur.Timestamp.Local().Format("2006-01-02 15:04")
}
