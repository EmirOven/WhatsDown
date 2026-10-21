package ui

import (
	"context"
	"log"

	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"postbetasoftware.com/whatsdown/internal/store"
)

func (c *MainController) deleteChatLocally(chat store.Chat) {
	if c.store == nil {
		return
	}
	go func() {
		if err := c.store.DeleteChatLocally(context.Background(), chat.JID); err != nil {
			log.Printf("deleting chat %s locally: %v", chat.JID, err)
			return
		}
		glib.IdleAdd(func() {
			if c.currentChat != nil && c.isSameChat(c.currentChat.JID, chat.JID) {
				c.currentChat = nil
				if c.contentStack != nil {
					c.contentStack.SetVisibleChildName("empty_chat_page")
				}
				if c.splitView != nil {
					c.splitView.SetShowContent(false)
				}
			}
			c.loadChats()
			c.loadArchivedChats()
		})
	}()
}
