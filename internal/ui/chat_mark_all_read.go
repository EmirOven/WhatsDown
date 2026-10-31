package ui

import (
	"context"
	"errors"
	"fmt"
	"log"
	"postbetasoftware.com/whatsdown/internal/wasync"
	"time"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"go.mau.fi/whatsmeow/appstate"
	"go.mau.fi/whatsmeow/types"
	"postbetasoftware.com/whatsdown/internal/store"
)

type readSyncClient interface {
	SendAppState(context.Context, appstate.PatchInfo) error
	MarkRead(context.Context, []types.MessageID, time.Time, types.JID, types.JID, ...types.ReceiptType) error
}

func markAllUnreadChats(ctx context.Context, s *store.MessageStore, client readSyncClient, at time.Time) error {
	if s == nil {
		return nil
	}
	if client == nil && !s.IsDemo() {
		return fmt.Errorf("connect to WhatsApp to mark all chats as read")
	}
	chats, err := s.ListChats(ctx)
	if err != nil {
		return err
	}
	archived, err := s.ListArchivedChats(ctx)
	if err != nil {
		return err
	}
	var failures []error
	for _, chat := range append(chats, archived...) {
		if chat.UnreadCount == 0 {
			continue
		}
		if err := markUnreadChat(ctx, s, client, chat, at); err != nil {
			failures = append(failures, fmt.Errorf("%s: %w", chat.Name, err))
		}
	}
	return errors.Join(failures...)
}

func markUnreadChat(ctx context.Context, s *store.MessageStore, client readSyncClient, chat store.Chat, at time.Time) error {
	jid, err := types.ParseJID(chat.JID)
	if err != nil {
		return err
	}
	if client != nil && !s.IsDemo() {
		appJID := jid
		if jid.Server == types.HiddenUserServer {
			if pn := s.ResolveLIDtoPN(ctx, chat.JID); pn != "" {
				appJID, _ = types.ParseJID(pn)
			}
		}
		messages, err := s.ReadReceiptMessages(ctx, chat.JID, at, chat.UnreadCount)
		if err != nil {
			return err
		}
		senders := make(map[types.JID][]types.MessageID)
		for _, m := range messages {
			sender, err := types.ParseJID(m.SenderJID)
			if err != nil || sender.IsEmpty() {
				if jid.Server == types.GroupServer {
					continue
				}
				sender = jid
			}
			senders[sender] = append(senders[sender], m.ID)
		}
		for sender, ids := range senders {
			for len(ids) > 0 {
				n := min(200, len(ids))
				if err := client.MarkRead(ctx, ids[:n], at, jid, sender); err != nil {
					return err
				}
				ids = ids[n:]
			}
		}
		if err := wasync.SendAppState(ctx, client, appstate.BuildMarkChatAsRead(appJID, true, at, nil)); err != nil {
			return err
		}
	}
	return s.MarkChatReadAt(ctx, jid, at)
}

func (c *MainController) updateMarkAllReadButton() {
	if c.markAllReadBtn == nil {
		return
	}
	unread := false
	for _, ch := range c.chats {
		unread = unread || ch.UnreadCount > 0
	}
	for _, ch := range c.archivedChats {
		unread = unread || ch.UnreadCount > 0
	}
	c.markAllReadBtn.SetSensitive(unread && !c.markingAllRead)
}

func (c *MainController) markAllChatsRead() {
	if c.markingAllRead || c.store == nil {
		return
	}
	c.markingAllRead = true
	c.updateMarkAllReadButton()
	at := time.Now()
	var client readSyncClient
	if c.client != nil && c.client.IsConnected() {
		client = c.client
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		err := markAllUnreadChats(ctx, c.store, client, at)
		glib.IdleAdd(func() {
			c.markingAllRead = false
			c.withdrawAllNotifications()
			c.loadChats()
			c.loadArchivedChats()
			c.updateMarkAllReadButton()
			if err != nil {
				log.Printf("marking all chats read: %v", err)
				alert := adw.NewAlertDialog("Could Not Mark All Chats as Read", "Some chats could not be synced with WhatsApp. Check your connection and try again.")
				alert.AddResponse("close", "Close")
				alert.Present(c.window)
			}
		})
	}()
}
