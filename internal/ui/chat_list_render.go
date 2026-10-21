package ui

import (
	"strings"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/store"
)

func (c *MainController) renderChatList(chats []store.Chat, isArchived bool) {
	listBox := c.chatsList
	searchFilter := c.chatsSearchFilter
	if isArchived {
		listBox = c.archivedChatsList
		searchFilter = c.archivedSearchFilter
	}
	if listBox == nil {
		return
	}
	listBox.RemoveAll()

	if isArchived && len(chats) == 0 {
		c.renderArchivedEmptyState(listBox)
		return
	}

	searchLower := strings.ToLower(strings.TrimSpace(searchFilter))
	topLevel, childrenMap := groupCommunities(chats, searchLower, c.expandedCommunities)
	var displayedRows []chatRowItem

	for _, item := range topLevel {
		if item.IsParent {
			c.appendCommunityRows(listBox, item, childrenMap[item.JID], searchLower, isArchived, &displayedRows)
		} else {
			if searchLower != "" && !strings.Contains(strings.ToLower(item.Name), searchLower) && !strings.Contains(strings.ToLower(item.LastMessageText), searchLower) {
				continue
			}
			row := c.createChatRow(*item, isArchived)
			listBox.Append(row)
			displayedRows = append(displayedRows, chatRowItem{chat: *item})
			if c.currentChat != nil && c.isSameChat(c.currentChat.JID, item.JID) {
				listBox.SelectRow(row)
			}
		}
	}

	if isArchived {
		c.displayedArchivedRows = displayedRows
	} else {
		c.displayedChatRows = displayedRows
	}
}

func (c *MainController) renderArchivedEmptyState(listBox *gtk.ListBox) {
	c.displayedArchivedRows = nil
	emptyRow := gtk.NewListBoxRow()
	emptyRow.SetSelectable(false)
	emptyRow.SetActivatable(false)
	box := gtk.NewBox(gtk.OrientationVertical, 8)
	box.SetMarginTop(32)
	box.SetMarginBottom(32)
	box.SetHAlign(gtk.AlignCenter)

	icon := gtk.NewImageFromIconName("folder-download-symbolic")
	icon.SetPixelSize(36)
	icon.AddCSSClass("dim-label")
	box.Append(icon)

	lbl := gtk.NewLabel("No archived chats")
	lbl.AddCSSClass("dim-label")
	box.Append(lbl)

	emptyRow.SetChild(box)
	listBox.Append(emptyRow)
}

func (c *MainController) isSameChat(j1, j2 string) bool {
	return j1 == j2 || (c.isSelfJID(j1) && c.isSelfJID(j2))
}
