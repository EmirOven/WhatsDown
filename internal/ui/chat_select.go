package ui

import (
	"context"
	"strings"

	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"go.mau.fi/whatsmeow/types"
	"postbetasoftware.com/whatsdown/internal/avatar"
	"postbetasoftware.com/whatsdown/internal/store"
)

func (c *MainController) selectChat(chat store.Chat) {
	c.cancelMessageSelection()
	c.currentChat = &chat
	c.setTypingIndicator(false)

	displayName := formatDisplayName(chat)
	if c.activeChatBtn != nil {
		c.activeChatBtn.SetSensitive(true)
	}
	if c.activeChatTitle != nil {
		c.activeChatTitle.SetTitle(displayName)
	}
	c.updateTopbarSubtitle()
	if c.activeChatAvatar != nil {
		avatar.LoadAvatar(c.client, chat.JID, displayName, c.activeChatAvatar)
	}
	if c.contentStack != nil {
		c.contentStack.SetVisibleChildName("chat")
	}

	if c.client != nil && !strings.HasSuffix(chat.JID, "@g.us") && !strings.HasSuffix(chat.JID, "@broadcast") && !strings.HasSuffix(chat.JID, "@newsletter") {
		go func() {
			if targetJID, err := types.ParseJID(chat.JID); err == nil {
				_ = c.client.SubscribePresence(context.Background(), targetJID)
			}
		}()
	}

	if c.chatSearchBtn != nil {
		c.chatSearchBtn.SetSensitive(true)
		c.chatSearchBtn.SetActive(false)
	}
	if c.chatSearchBar != nil {
		c.chatSearchBar.SetSearchMode(false)
	}
	if c.chatSearchEntry != nil {
		c.chatSearchEntry.SetText("")
	}
	c.messagesSearchFilter = ""
	c.searchMatchIndices = nil
	c.currentSearchMatchIndex = 0
	c.updateSearchUI()

	if c.splitView != nil {
		c.splitView.SetShowContent(true)
	}
	c.cancelReply()
	c.hideMentionPopup()
	activeMentionState.activeMembers = nil
	activeMentionState.insertedMentions = make(map[string]string)
	c.deleteVoiceRecording()
	c.applyChatWallpaper(chat.JID)

	c.markChatRead(chat.JID)
	glib.IdleAdd(func() {
		c.loadChats()
		c.loadArchivedChats()
	})

	if c.messagesList != nil {
		c.loadMessages(chat.JID)
	}
}

func (c *MainController) markChatRead(chatJIDStr string) {
	if chatJIDStr == "" {
		return
	}
	c.withdrawChatNotification(chatJIDStr)
	targetJID, err := types.ParseJID(chatJIDStr)
	if err != nil {
		return
	}

	if c.store != nil {
		_ = c.store.MarkChatRead(context.Background(), targetJID)
	}

	for i := range c.chats {
		if c.chats[i].JID == chatJIDStr {
			c.chats[i].UnreadCount = 0
			break
		}
	}
	for i := range c.archivedChats {
		if c.archivedChats[i].JID == chatJIDStr {
			c.archivedChats[i].UnreadCount = 0
			break
		}
	}

	if c.client != nil {
		go c.sendReadReceipts(targetJID, chatJIDStr)
	}
}
