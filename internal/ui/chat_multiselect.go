package ui

import (
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/store"
)

func (c *MainController) chatListRows(archived bool) []chatRowItem {
	if archived {
		return c.displayedArchivedRows
	}
	return c.displayedChatRows
}

func (c *MainController) bindChatListSelection(list *gtk.ListBox, archived bool) {
	if list == nil {
		return
	}
	list.SetSelectionMode(gtk.SelectionMultiple)
	list.SetActivateOnSingleClick(true)
	click := gtk.NewGestureClick()
	click.SetPropagationPhase(gtk.PhaseCapture)
	click.SetButton(gdk.BUTTON_PRIMARY)
	c.trackWidgetSignal(list, click, click.ConnectPressed(func(_ int, _ float64, y float64) {
		c.chatControlPressed = click.CurrentEventState()&gdk.ControlMask != 0
		c.chatDragSelection = nil
		if !c.chatControlPressed && len(c.selectedChatJIDs) > 1 {
			row := list.RowAtY(int(y))
			items := c.chatListRows(archived)
			if row != nil && row.Index() >= 0 && row.Index() < len(items) && c.selectedChatJIDs[items[row.Index()].chat.JID] {
				c.chatDragSelection = make(map[string]bool, len(c.selectedChatJIDs))
				for jid := range c.selectedChatJIDs {
					c.chatDragSelection[jid] = true
				}
			}
		}
	}))
	list.AddController(click)
	keys := gtk.NewEventControllerKey()
	keys.SetPropagationPhase(gtk.PhaseCapture)
	c.trackWidgetSignal(list, keys, keys.ConnectKeyPressed(func(_ uint, _ uint, state gdk.ModifierType) bool {
		c.chatControlPressed = state&gdk.ControlMask != 0
		return false
	}))
	list.AddController(keys)
	c.trackWidgetSignal(list, list, list.ConnectSelectedRowsChanged(func() {
		if c.suppressChatSelection {
			return
		}
		// Keep one selection set across the two sidebar pages, preserving it over
		// sync-driven list rebuilds without confusing selection with activation.
		if c.selectedChatList != nil && c.selectedChatList != list {
			c.suppressChatSelection = true
			c.selectedChatList.UnselectAll()
			c.suppressChatSelection = false
		}
		c.selectedChatList = list
		c.selectedChatJIDs = make(map[string]bool)
		rows := c.chatListRows(archived)
		for _, row := range list.SelectedRows() {
			i := row.Index()
			if i >= 0 && i < len(rows) && !rows[i].isFolderHeader {
				c.selectedChatJIDs[rows[i].chat.JID] = true
			}
		}
	}))
	c.trackWidgetSignal(list, list, list.ConnectRowActivated(func(row *gtk.ListBoxRow) {
		if row == nil || c.suppressChatSelection || c.chatControlPressed {
			return
		}
		rows := c.chatListRows(archived)
		i := row.Index()
		if i < 0 || i >= len(rows) {
			return
		}
		item := rows[i]
		if item.isLooseDropZone {
			return
		}
		if item.isFolderHeader {
			c.toggleChatFolder(item.folderID)
			return
		}
		if item.isCommunityHeader {
			c.toggleCommunityExpanded(item.communityJID)
			return
		}
		c.selectOnlyChatRow(row, item.chat, archived)
		if c.currentChat == nil || !c.isSameChat(c.currentChat.JID, item.chat.JID) {
			c.selectChat(item.chat)
		}
	}))
	c.bindChatListDrop(list, archived)
}

func (c *MainController) selectChatForContext(row *gtk.ListBoxRow, chat store.Chat, archived bool) {
	if c.selectedChatJIDs[chat.JID] {
		return
	}
	c.selectOnlyChatRow(row, chat, archived)
}

// Activation replaces the sidebar selection before chat loading can rebuild rows.
func (c *MainController) selectOnlyChatRow(row *gtk.ListBoxRow, chat store.Chat, archived bool) {
	list := c.chatsList
	if archived {
		list = c.archivedChatsList
	}
	if list == nil {
		return
	}
	wasSuppressed := c.suppressChatSelection
	c.suppressChatSelection = true
	defer func() { c.suppressChatSelection = wasSuppressed }()
	if c.selectedChatList != nil {
		c.selectedChatList.UnselectAll()
	}
	list.UnselectAll()
	list.SelectRow(row)
	c.selectedChatJIDs = map[string]bool{chat.JID: true}
	c.selectedChatList = list
}

func (c *MainController) selectedFolderChats(chat store.Chat) []string {
	if !c.selectedChatJIDs[chat.JID] {
		return []string{chat.JID}
	}
	// Preserve visible order when moving several selected chats together.
	var jids []string
	seen := make(map[string]bool)
	for _, items := range [][]chatRowItem{c.displayedChatRows, c.displayedArchivedRows} {
		for _, item := range items {
			jid := item.chat.JID
			if !item.isFolderHeader && c.selectedChatJIDs[jid] && !seen[jid] {
				jids = append(jids, jid)
				seen[jid] = true
			}
		}
	}
	// Selected children may be hidden by a collapsed folder or a search filter.
	for _, chats := range [][]store.Chat{c.chats, c.archivedChats} {
		for _, item := range chats {
			if c.selectedChatJIDs[item.JID] && !seen[item.JID] {
				jids = append(jids, item.JID)
				seen[item.JID] = true
			}
		}
	}
	return jids
}
