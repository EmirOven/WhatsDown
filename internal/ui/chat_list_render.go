package ui

import (
	"context"
	"log"
	"strings"

	coreglib "github.com/diamondburned/gotk4/pkg/core/glib"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/store"
)

func (c *MainController) renderChatList(chats []store.Chat, isArchived bool) {
	// Keep GTK's source and destination widgets alive until a drag finishes.
	if c.sidebarDrag != nil {
		return
	}
	wasSuppressed := c.suppressChatSelection
	c.suppressChatSelection = true
	defer func() { c.suppressChatSelection = wasSuppressed }()
	listBox := c.chatsList
	searchFilter := c.chatsSearchFilter
	if isArchived {
		listBox = c.archivedChatsList
		searchFilter = c.archivedSearchFilter
	}
	if listBox == nil {
		return
	}
	c.clearChatDropIndicator()
	// Only release menus belonging to this list (active and archived lists
	// are refreshed independently).
	for row, cleanup := range c.chatMenuCleanups {
		if parent := row.Parent(); parent != nil && coreglib.BaseObject(parent).Native() == coreglib.BaseObject(listBox).Native() {
			cleanup()
			delete(c.chatMenuCleanups, row)
		}
	}
	for child := listBox.FirstChild(); child != nil; child = gtk.BaseWidget(child).NextSibling() {
		c.releaseMessageWidget(child)
	}
	listBox.RemoveAll()

	if isArchived && len(chats) == 0 {
		c.renderArchivedEmptyState(listBox)
		return
	}

	if c.store != nil {
		folders, err := c.store.ListChatFolders(context.Background())
		if err != nil {
			log.Printf("loading chat folders: %v", err)
		} else {
			c.chatFolders = folders
			for i := range c.chatFolders {
				if value, ok := c.folderCollapsedOverrides[c.chatFolders[i].ID]; ok {
					c.chatFolders[i].Collapsed = value
				}
			}
		}
	}
	searchLower := strings.ToLower(strings.TrimSpace(searchFilter))
	var displayedRows []chatRowItem
	chats = c.appendChatFolderRows(listBox, chats, searchLower, isArchived, &displayedRows)
	topLevel, childrenMap := groupCommunities(chats, searchLower, c.expandedCommunities)

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
		}
	}

	for i, item := range displayedRows {
		row := listBox.RowAtIndex(i)
		c.bindChatRowDrag(row, item, isArchived)
		if !item.isFolderHeader && !item.isLooseDropZone && (c.selectedChatJIDs[item.chat.JID] || (len(c.selectedChatJIDs) == 0 && c.currentChat != nil && c.isSameChat(c.currentChat.JID, item.chat.JID))) {
			listBox.SelectRow(row)
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
	if j1 == j2 || (c.isSelfJID(j1) && c.isSelfJID(j2)) {
		return true
	}
	if c.store != nil {
		p1, p2 := j1, j2
		if pn := c.store.ResolveLIDtoPN(context.Background(), j1); pn != "" {
			p1 = pn
		}
		if pn := c.store.ResolveLIDtoPN(context.Background(), j2); pn != "" {
			p2 = pn
		}
		return p1 == p2
	}
	return false
}
