package ui

import (
	"sort"
	"strings"

	"postbetasoftware.com/whatsdown/internal/store"
)

func groupCommunities(chats []store.Chat, searchLower string, expanded map[string]bool) ([]*store.Chat, map[string][]store.Chat) {
	communityMap := make(map[string]*store.Chat)
	childrenMap := make(map[string][]store.Chat)

	for _, chat := range chats {
		if chat.IsParent {
			cp := chat
			communityMap[chat.JID] = &cp
		}
		if chat.ParentJID != "" {
			childrenMap[chat.ParentJID] = append(childrenMap[chat.ParentJID], chat)
			if _, exists := communityMap[chat.ParentJID]; !exists {
				communityMap[chat.ParentJID] = &store.Chat{JID: chat.ParentJID, Name: "Community", IsParent: true}
			}
		}
	}

	for pJID, comm := range communityMap {
		children := childrenMap[pJID]
		for _, ch := range children {
			comm.UnreadCount += ch.UnreadCount
			if ch.LastMessageAt.After(comm.LastMessageAt) {
				comm.LastMessageAt = ch.LastMessageAt
				comm.LastMessageText = ch.Name + ": " + ch.LastMessageText
			}
		}
	}

	if searchLower != "" {
		for pJID, children := range childrenMap {
			for _, ch := range children {
				if strings.Contains(strings.ToLower(ch.Name), searchLower) || strings.Contains(strings.ToLower(ch.LastMessageText), searchLower) {
					expanded[pJID] = true
					break
				}
			}
		}
	}

	topLevel := make([]*store.Chat, 0, len(chats))
	addedCommunities := make(map[string]bool)

	for i := range chats {
		chat := &chats[i]
		if chat.ParentJID != "" {
			if !addedCommunities[chat.ParentJID] {
				if comm := communityMap[chat.ParentJID]; comm != nil {
					topLevel = append(topLevel, comm)
					addedCommunities[chat.ParentJID] = true
				}
			}
			continue
		}
		if chat.IsParent {
			if !addedCommunities[chat.JID] {
				if comm := communityMap[chat.JID]; comm != nil {
					topLevel = append(topLevel, comm)
					addedCommunities[chat.JID] = true
				}
			}
			continue
		}
		topLevel = append(topLevel, chat)
	}

	sort.SliceStable(topLevel, func(i, j int) bool {
		if topLevel[i].Pinned != topLevel[j].Pinned {
			return topLevel[i].Pinned
		}
		return topLevel[i].LastMessageAt.After(topLevel[j].LastMessageAt)
	})

	return topLevel, childrenMap
}
