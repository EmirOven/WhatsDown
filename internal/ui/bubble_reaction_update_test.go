package ui

import (
	"context"
	"testing"
	"time"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/store"
)

func TestReactionPillOnLoadUpdateAndRemoval(t *testing.T) {
	c := newFolderSidebar(t)
	ctx := context.Background()
	chat := store.Chat{JID: "0@a.test"}
	c.currentChat = &chat
	c.messagesList = gtk.NewListBox()
	t.Cleanup(func() {
		for child := c.messagesList.FirstChild(); child != nil; child = gtk.BaseWidget(child).NextSibling() {
			c.releaseMessageWidget(child)
		}
		c.messagesList.RemoveAll()
	})
	msg := store.StoredMessage{ChatJID: chat.JID, ID: "reacted", Kind: store.MessageKindText, Text: "hello", Timestamp: time.Now()}
	if err := c.store.SaveMessage(ctx, msg); err != nil {
		t.Fatal(err)
	}
	reaction := store.Reaction{ChatJID: chat.JID, MessageID: msg.ID, SenderJID: "0@b.test", Emoji: "👍", Timestamp: time.Now()}
	if err := c.store.SaveReaction(ctx, reaction); err != nil {
		t.Fatal(err)
	}
	c.loadMessages(chat.JID)
	var findPill func(gtk.Widgetter) *gtk.Widget
	findPill = func(widget gtk.Widgetter) *gtk.Widget {
		w := gtk.BaseWidget(widget)
		if w.HasCSSClass("reaction-pill") {
			return w
		}
		for child := w.FirstChild(); child != nil; child = gtk.BaseWidget(child).NextSibling() {
			if found := findPill(child); found != nil {
				return found
			}
		}
		return nil
	}
	if findPill(c.messageRowMap[msg.ID]) == nil {
		t.Fatal("stored reaction not displayed on load")
	}
	reaction.Emoji = "❤️"
	if err := c.store.SaveReaction(ctx, reaction); err != nil {
		t.Fatal(err)
	}
	c.UpdateMessageReactions(chat.JID, msg.ID)
	if findPill(c.messageRowMap[msg.ID]) == nil || c.currentMessages[0].Reactions[0].Emoji != "❤️" {
		t.Fatal("live reaction update missing")
	}
	if err := c.store.RemoveReaction(ctx, chat.JID, msg.ID, reaction.SenderJID); err != nil {
		t.Fatal(err)
	}
	c.UpdateMessageReactions(chat.JID, msg.ID)
	if findPill(c.messageRowMap[msg.ID]) != nil {
		t.Fatal("reaction pill remains after removal")
	}
}
