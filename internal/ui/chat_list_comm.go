package ui

import (
	"sort"
	"strings"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/store"
)

func (c *MainController) appendCommunityRows(listBox *gtk.ListBox, item *store.Chat, children []store.Chat, searchLower string, isArchived bool, rows *[]chatRowItem) {
	sort.SliceStable(children, func(i, j int) bool {
		if children[i].IsDefaultSubGroup != children[j].IsDefaultSubGroup {
			return children[i].IsDefaultSubGroup
		}
		return children[i].LastMessageAt.After(children[j].LastMessageAt)
	})

	commMatches := searchLower == "" || strings.Contains(strings.ToLower(item.Name), searchLower)
	hasMatchingChild := false
	if searchLower != "" {
		for _, ch := range children {
			if strings.Contains(strings.ToLower(ch.Name), searchLower) || strings.Contains(strings.ToLower(ch.LastMessageText), searchLower) {
				hasMatchingChild = true
				break
			}
		}
	}
	if searchLower != "" && !commMatches && !hasMatchingChild {
		return
	}

	isExpanded := c.expandedCommunities[item.JID]
	commRow := c.createCommunityHeaderRow(*item, len(children), isExpanded)
	listBox.Append(commRow)
	*rows = append(*rows, chatRowItem{isCommunityHeader: true, communityJID: item.JID, chat: *item})

	if isExpanded {
		for _, child := range children {
			if searchLower != "" && !commMatches && !strings.Contains(strings.ToLower(child.Name), searchLower) && !strings.Contains(strings.ToLower(child.LastMessageText), searchLower) {
				continue
			}
			childRow := c.createCommunityChildRow(child, isArchived)
			listBox.Append(childRow)
			*rows = append(*rows, chatRowItem{isCommunityHeader: false, communityJID: item.JID, chat: child})
			if c.currentChat != nil && c.isSameChat(c.currentChat.JID, child.JID) {
				listBox.SelectRow(childRow)
			}
		}
	}
}
