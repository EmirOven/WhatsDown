package ui

import (
	"context"
	"testing"
	"time"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"go.mau.fi/whatsmeow/proto/waWeb"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"postbetasoftware.com/whatsdown/internal/store"
)

func TestGroupReceiptTicksReflectAllRecipients(t *testing.T) {
	c := newFolderSidebar(t)
	ctx := context.Background()
	chat := store.Chat{JID: "123@g.us"}
	c.currentChat = &chat
	c.messagesList = gtk.NewListBox()
	t.Cleanup(func() {
		for child := c.messagesList.FirstChild(); child != nil; child = gtk.BaseWidget(child).NextSibling() {
			c.releaseMessageWidget(child)
		}
		c.messagesList.RemoveAll()
	})
	msg := store.StoredMessage{ChatJID: chat.JID, ID: "ticks", FromMe: true, Kind: store.MessageKindText, Text: "hello", Status: store.MessageStatusSent, Timestamp: time.Now()}
	if err := c.store.SaveMessage(ctx, msg); err != nil {
		t.Fatal(err)
	}
	recipients := []string{"1@s.whatsapp.net", "2@s.whatsapp.net"}
	if err := c.store.SetMessageRecipients(ctx, chat.JID, msg.ID, recipients); err != nil {
		t.Fatal(err)
	}
	c.loadMessages(chat.JID)
	check := func(text string, read bool) {
		t.Helper()
		label := c.messageStatusLabels[msg.ID]
		if label == nil || label.Text() != text || label.HasCSSClass("chat-bubble-status-read") != read {
			t.Fatalf("wrong tick label: %v", label)
		}
	}
	update := func(recipient string, status int) {
		t.Helper()
		if err := c.store.SaveMessageReceipt(ctx, chat.JID, []string{msg.ID}, recipient, status, time.Now()); err != nil {
			t.Fatal(err)
		}
		c.RefreshReceiptMessages(chat.JID, []string{msg.ID})
	}
	check("✓", false)
	update(recipients[0], store.MessageStatusRead)
	check("✓", false)
	update(recipients[1], store.MessageStatusDelivered)
	check("✓✓", false)
	update(recipients[1], store.MessageStatusRead)
	check("✓✓", true)
}

func TestUnknownStatusDoesNotShowBlueTicks(t *testing.T) {
	c := newFolderSidebar(t)
	label := gtk.NewLabel("")
	c.applyStatusToLabel(label, store.MessageStatusNone)
	if label.HasCSSClass("chat-bubble-status-read") || label.Text() != "✓" {
		t.Fatal("unknown status displayed as read")
	}
}

func TestMessageInfoRefreshesOnPhoneMetadata(t *testing.T) {
	c := dialogTestController(t)
	ctx := context.Background()
	chat, id := "123@g.us", "phone-info"
	msg := store.StoredMessage{ChatJID: chat, ID: id, FromMe: true, Kind: store.MessageKindText, Text: "hello", Timestamp: time.Now()}
	if err := c.store.SaveMessage(ctx, msg); err != nil {
		t.Fatal(err)
	}
	recipients := []string{"1@s.whatsapp.net", "2@s.whatsapp.net", "3@s.whatsapp.net", "4@s.whatsapp.net", "5@s.whatsapp.net"}
	if err := c.store.SetMessageRecipients(ctx, chat, id, recipients); err != nil {
		t.Fatal(err)
	}
	view := c.buildMessageInfo(msg)
	t.Cleanup(view.window.Destroy)
	view.window.Present()
	check := func(title string) {
		t.Helper()
		found := false
		walkDialog(view.window, func(widget gtk.Widgetter) {
			if group, ok := widget.(*adw.PreferencesGroup); ok && group.Title() == title {
				found = true
			}
		})
		if !found {
			t.Fatalf("missing message info section %q", title)
		}
	}
	check("Pending Delivery · 5")
	if err := c.store.SaveMessageReceipt(ctx, chat, []string{id}, recipients[0], store.MessageStatusRead, time.Unix(200, 0)); err != nil {
		t.Fatal(err)
	}
	// A phone response refreshes the open dialog immediately, before the timer.
	c.HandleIncomingLiveMessage(&events.Message{Info: types.MessageInfo{
		MessageSource: types.MessageSource{Chat: types.NewJID("123", types.GroupServer), IsFromMe: true}, ID: id,
	}, SourceWebMsg: &waWeb.WebMessageInfo{}})
	check("Read by · 1")
	check("Pending Delivery · 4")
	view.window.Close()
	if len(c.messageInfoViews) != 0 {
		t.Fatal("closed message info view retained")
	}
}
