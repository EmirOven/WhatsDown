package ui

import (
	"context"
	"strings"
	"sync"

	"postbetasoftware.com/whatsdown/internal/markdown"
	"postbetasoftware.com/whatsdown/internal/store"
)

var (
	contactNameMu    sync.RWMutex
	contactNameCache = make(map[string]string)
)

func (c *MainController) resolveContactName(targetJID string) string {
	if targetJID == "" {
		return ""
	}
	contactNameMu.RLock()
	if name, ok := contactNameCache[targetJID]; ok {
		contactNameMu.RUnlock()
		return name
	}
	contactNameMu.RUnlock()

	for _, ch := range c.chats {
		if ch.JID == targetJID {
			disp := formatDisplayName(ch)
			contactNameMu.Lock()
			contactNameCache[targetJID] = disp
			contactNameMu.Unlock()
			return disp
		}
	}
	for _, ch := range c.archivedChats {
		if ch.JID == targetJID {
			disp := formatDisplayName(ch)
			contactNameMu.Lock()
			contactNameCache[targetJID] = disp
			contactNameMu.Unlock()
			return disp
		}
	}

	if c.store != nil {
		chat, err := c.store.GetChatOrContact(context.Background(), targetJID)
		if err == nil {
			disp := formatDisplayName(chat)
			contactNameMu.Lock()
			contactNameCache[targetJID] = disp
			contactNameMu.Unlock()
			return disp
		}
	}
	return cleanPhoneNumber(targetJID)
}

func (c *MainController) openUserDetails(targetJID string) {
	if targetJID == "" {
		return
	}
	if !strings.Contains(targetJID, "@") && markdown.IsAllDigits(targetJID) && len(targetJID) >= 5 {
		targetJID = targetJID + "@s.whatsapp.net"
	}

	for _, ch := range c.chats {
		if ch.JID == targetJID {
			c.showDetailsAndMedia(ch)
			return
		}
	}
	for _, ch := range c.archivedChats {
		if ch.JID == targetJID {
			c.showDetailsAndMedia(ch)
			return
		}
	}

	if c.store != nil {
		chat, err := c.store.GetChatOrContact(context.Background(), targetJID)
		if err == nil {
			c.showDetailsAndMedia(chat)
			return
		}
	}

	c.showDetailsAndMedia(store.Chat{
		JID:  targetJID,
		Name: cleanPhoneNumber(targetJID),
	})
}
