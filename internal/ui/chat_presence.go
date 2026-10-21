package ui

import (
	"strings"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

func (c *MainController) isContactOnline(jid string) bool {
	if c.onlineContacts == nil {
		return false
	}
	if c.isSelfJID(jid) {
		return true
	}
	if c.onlineContacts[jid] {
		return true
	}
	for contactJID, online := range c.onlineContacts {
		if online && c.isSameChat(contactJID, jid) {
			return true
		}
	}
	return false
}

func (c *MainController) updateTopbarSubtitle() {
	if c.currentChat == nil || c.activeChatTitle == nil {
		return
	}
	isDM := !strings.HasSuffix(c.currentChat.JID, "@g.us") && !strings.HasSuffix(c.currentChat.JID, "@broadcast") && !strings.HasSuffix(c.currentChat.JID, "@newsletter")
	if isDM && c.isContactOnline(c.currentChat.JID) {
		c.activeChatTitle.SetSubtitle("Online")
		return
	}
	c.activeChatTitle.SetSubtitle(formatTopbarSubtitle(*c.currentChat))
}

func (c *MainController) HandlePresence(evt *events.Presence) {
	if evt == nil {
		return
	}
	jidStr := evt.From.ToNonAD().String()
	if c.onlineContacts == nil {
		c.onlineContacts = make(map[string]bool)
	}
	c.onlineContacts[jidStr] = !evt.Unavailable
	if c.currentChat != nil && c.isSameChat(c.currentChat.JID, jidStr) {
		c.updateTopbarSubtitle()
	}
}

func (c *MainController) HandleChatPresence(evt *events.ChatPresence) {
	if c.currentChat == nil || evt == nil {
		return
	}
	if evt.IsGroup || strings.HasSuffix(c.currentChat.JID, "@g.us") || strings.HasSuffix(c.currentChat.JID, "@broadcast") || strings.HasSuffix(c.currentChat.JID, "@newsletter") {
		return
	}

	chatJID := evt.Chat.ToNonAD().String()
	if chatJID == "" {
		chatJID = evt.Sender.ToNonAD().String()
	}
	if !c.isSameChat(c.currentChat.JID, chatJID) {
		return
	}

	isComposing := evt.State == types.ChatPresenceComposing
	if isComposing {
		if c.onlineContacts == nil {
			c.onlineContacts = make(map[string]bool)
		}
		c.onlineContacts[chatJID] = true
		c.updateTopbarSubtitle()
	}
	c.setTypingIndicator(isComposing)
}
