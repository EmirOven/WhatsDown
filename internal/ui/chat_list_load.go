package ui

import (
	"context"
	"log"
)

func (c *MainController) LoadChats() {
	if c.store == nil {
		return
	}
	c.suppressChatSelection = true
	defer func() {
		c.suppressChatSelection = false
	}()

	chats, err := c.store.ListChats(context.Background())
	if err != nil {
		log.Printf("loading chats: %v", err)
		return
	}
	c.chats = chats
	c.updateMarkAllReadButton()
	c.renderChatList(chats, false)
}

func (c *MainController) loadChats() {
	c.LoadChats()
}

func (c *MainController) LoadArchivedChats() {
	if c.store == nil {
		return
	}
	c.suppressChatSelection = true
	defer func() {
		c.suppressChatSelection = false
	}()

	chats, err := c.store.ListArchivedChats(context.Background())
	if err != nil {
		log.Printf("loading archived chats: %v", err)
		return
	}
	c.archivedChats = chats
	c.updateMarkAllReadButton()
	c.renderChatList(chats, true)
}

func (c *MainController) loadArchivedChats() {
	c.LoadArchivedChats()
}

func (c *MainController) toggleCommunityExpanded(communityJID string) {
	c.expandedCommunities[communityJID] = !c.expandedCommunities[communityJID]
	c.LoadChats()
	c.LoadArchivedChats()
}
