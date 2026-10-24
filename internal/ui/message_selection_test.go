package ui

import (
	"context"
	"testing"
	"time"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/store"
)

func TestInlineMessageSelectionSurvivesUpdatesAndCancels(t *testing.T) {
	c := newFolderSidebar(t)
	ctx := context.Background()
	chat := store.Chat{JID: "0@a.test"}
	c.currentChat = &chat
	c.messagesList = gtk.NewListBox()
	c.chatPage = gtk.NewBox(gtk.OrientationVertical, 0)
	c.chatPage.Append(c.messagesList)
	c.chatInputStack = gtk.NewStack()
	c.chatPage.Append(c.chatInputStack)
	t.Cleanup(func() {
		c.releaseMessageWidget(c.chatPage)
		c.messagesList.RemoveAll()
	})
	for i, id := range []string{"one", "two", "three"} {
		if err := c.store.SaveMessage(ctx, store.StoredMessage{ChatJID: chat.JID, ID: id, Kind: store.MessageKindText, Text: id, Timestamp: time.Unix(int64(100+i), 0)}); err != nil {
			t.Fatal(err)
		}
	}
	c.loadMessages(chat.JID)
	windows := len(gtk.WindowListToplevels())
	c.showMessageSelection("two")
	state := c.messageSelection
	if len(gtk.WindowListToplevels()) != windows {
		t.Fatal("selection opened a modal")
	}
	if !state.active || !state.toolbar.Visible() || state.count.Text() != "1 selected" || c.chatInputStack.Visible() {
		t.Fatal("inline selection toolbar not shown correctly")
	}
	for id, check := range state.checks {
		if !check.Visible() || check.Active() != (id == "two") {
			t.Fatalf("incorrect checkbox state for %s", id)
		}
	}
	state.checks["one"].SetActive(true)
	selected := c.selectedMessages()
	if len(selected) != 2 || selected[0].ID != "one" || selected[1].ID != "two" {
		t.Fatalf("selected messages lost conversation order: %+v", selected)
	}
	c.UpdateSingleMessageRow("two")
	if !state.checks["two"].Active() || !c.messageRowMap["two"].HasCSSClass("message-selected") {
		t.Fatal("receipt update lost selection")
	}
	c.loadMessages(chat.JID)
	if len(c.selectedMessages()) != 2 {
		t.Fatal("reload lost selection")
	}
	if !c.handleChatViewKeyPress(gdk.KEY_a, 0, gdk.ControlMask) || len(c.selectedMessages()) != 3 {
		t.Fatal("Ctrl+A did not select messages")
	}
	if !c.handleChatViewKeyPress(gdk.KEY_Escape, 0, 0) {
		t.Fatal("Escape did not cancel selection")
	}
	if state.active || state.toolbar.Visible() || !c.chatInputStack.Visible() {
		t.Fatal("cancelling did not restore composer")
	}
	for id, check := range state.checks {
		if check.Visible() || c.messageRowMap[id].HasCSSClass("message-selected") {
			t.Fatal("selection decoration remained after cancelling")
		}
	}
	c.showMessageSelection("one")
	next := store.Chat{JID: "0@b.test"}
	c.currentChat = &next
	c.loadMessages(next.JID)
	if state.active || len(state.selected) != 0 || state.toolbar.Visible() {
		t.Fatal("selection leaked into another chat")
	}
}
