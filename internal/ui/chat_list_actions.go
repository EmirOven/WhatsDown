package ui

import (
	"context"
	"log"
	"postbetasoftware.com/whatsdown/internal/wasync"
	"strings"
	"time"

	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"go.mau.fi/whatsmeow/appstate"
	"go.mau.fi/whatsmeow/types"
	"postbetasoftware.com/whatsdown/internal/store"
)

func (c *MainController) togglePin(chat store.Chat) {
	if c.store == nil {
		return
	}
	newPinned := !chat.Pinned
	go func() {
		if err := c.store.SetChatPinned(context.Background(), chat.JID, newPinned); err != nil {
			log.Printf("toggling pin for %s: %v", chat.JID, err)
			return
		}
		if c.client != nil {
			targetStr := chat.JID
			if strings.HasSuffix(targetStr, "@lid") && c.store != nil {
				targetStr = c.store.ResolveLIDtoPN(context.Background(), targetStr)
			}
			if targetJID, err := types.ParseJID(targetStr); err == nil {
				_ = wasync.SendAppState(context.Background(), c.client, appstate.BuildPin(targetJID, newPinned))
			}
		}
		glib.IdleAdd(func() {
			c.loadChats()
			c.loadArchivedChats()
		})
	}()
}

func (c *MainController) toggleMute(chat store.Chat) {
	if c.store == nil {
		return
	}
	newMuted := !chat.Muted
	go func() {
		if err := c.store.SetChatMuted(context.Background(), chat.JID, newMuted); err != nil {
			log.Printf("toggling mute for %s: %v", chat.JID, err)
			return
		}
		if c.client != nil {
			targetStr := chat.JID
			if strings.HasSuffix(targetStr, "@lid") && c.store != nil {
				targetStr = c.store.ResolveLIDtoPN(context.Background(), targetStr)
			}
			if targetJID, err := types.ParseJID(targetStr); err == nil {
				_ = wasync.SendAppState(context.Background(), c.client, appstate.BuildMute(targetJID, newMuted, 0))
			}
		}
		glib.IdleAdd(func() {
			c.loadChats()
			c.loadArchivedChats()
		})
	}()
}

func (c *MainController) toggleArchive(chat store.Chat) {
	if c.store == nil {
		return
	}
	newArchived := !chat.Archived
	go func() {
		if err := c.store.SetChatArchived(context.Background(), chat.JID, newArchived); err != nil {
			log.Printf("toggling archive for %s: %v", chat.JID, err)
			return
		}
		if c.client != nil {
			targetStr := chat.JID
			if strings.HasSuffix(targetStr, "@lid") && c.store != nil {
				targetStr = c.store.ResolveLIDtoPN(context.Background(), targetStr)
			}
			if targetJID, err := types.ParseJID(targetStr); err == nil {
				_ = wasync.SendAppState(context.Background(), c.client, appstate.BuildArchive(targetJID, newArchived, time.Time{}, nil))
			}
		}
		glib.IdleAdd(func() {
			c.loadChats()
			c.loadArchivedChats()
		})
	}()
}
