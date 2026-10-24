package ui

import (
	"context"
	"encoding/json"
	"strings"

	coreglib "github.com/diamondburned/gotk4/pkg/core/glib"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

const chatDragPrefix = "whatsdown-chat-list:"

type chatDragPayload struct {
	FolderID string   `json:"folder,omitempty"`
	ChatJIDs []string `json:"chats,omitempty"`
}

func decodeChatDrag(value string) (chatDragPayload, bool) {
	var p chatDragPayload
	if !strings.HasPrefix(value, chatDragPrefix) || len(value) > 1<<20 {
		return p, false
	}
	if json.Unmarshal([]byte(strings.TrimPrefix(value, chatDragPrefix)), &p) != nil {
		return p, false
	}
	return p, (p.FolderID != "") != (len(p.ChatJIDs) != 0)
}

func (c *MainController) bindChatRowDrag(row *gtk.ListBoxRow, item chatRowItem, archived bool) {
	if row == nil || c.store == nil || item.isLooseDropZone {
		return
	}
	source := gtk.NewDragSource()
	source.SetActions(gdk.ActionMove)
	c.trackWidgetSignal(row, source, source.ConnectPrepare(func(_, _ float64) *gdk.ContentProvider {
		payload := chatDragPayload{}
		if item.isFolderHeader {
			payload.FolderID = item.folderID
		} else {
			if c.chatDragSelection[item.chat.JID] {
				c.selectedChatJIDs = c.chatDragSelection
				c.suppressChatSelection = true
				list := c.chatsList
				if archived {
					list = c.archivedChatsList
				}
				list.UnselectAll()
				for i, item := range c.chatListRows(archived) {
					if c.selectedChatJIDs[item.chat.JID] {
						list.SelectRow(list.RowAtIndex(i))
					}
				}
				c.suppressChatSelection = false
			}
			c.chatDragSelection = nil
			c.selectChatForContext(row, item.chat, archived)
			payload.ChatJIDs = c.selectedFolderChats(item.chat)
		}
		c.sidebarDrag = &payload
		encoded, _ := json.Marshal(payload)
		return gdk.NewContentProviderForValue(coreglib.NewValue(chatDragPrefix + string(encoded)))
	}))
	c.trackWidgetSignal(row, source, source.ConnectDragEnd(func(_ gdk.Dragger, _ bool) {
		c.sidebarDrag = nil
		c.clearChatDropIndicator()
		c.queueSidebarReload()
	}))
	row.AddController(source)
}

func (c *MainController) clearChatDropIndicator() {
	if c.sidebarDropRow != nil {
		for _, class := range []string{"folder-drop-target", "chat-drop-before", "chat-drop-after"} {
			c.sidebarDropRow.RemoveCSSClass(class)
		}
		c.sidebarDropRow = nil
	}
}

func (c *MainController) chatDropItem(list *gtk.ListBox, archived bool, y float64) (*gtk.ListBoxRow, chatRowItem, bool) {
	row := list.RowAtY(int(y))
	rows := c.chatListRows(archived)
	if row == nil || row.Index() < 0 || row.Index() >= len(rows) {
		return nil, chatRowItem{}, false
	}
	_, top, ok := row.TranslateCoordinates(list, 0, 0)
	return row, rows[row.Index()], ok && y-top > float64(row.Height())/2
}

func (c *MainController) bindChatListDrop(list *gtk.ListBox, archived bool) {
	target := gtk.NewDropTarget(coreglib.TypeString, gdk.ActionMove)
	c.trackWidgetSignal(list, target, target.ConnectMotion(func(_ float64, y float64) gdk.DragAction {
		c.clearChatDropIndicator()
		if c.sidebarDrag == nil {
			return 0
		}
		row, item, after := c.chatDropItem(list, archived, y)
		if row != nil {
			class := "folder-drop-target"
			if c.sidebarDrag.FolderID != "" || (item.folderID != "" && !item.isFolderHeader) {
				class = "chat-drop-before"
				if after {
					class = "chat-drop-after"
				}
			}
			row.AddCSSClass(class)
			c.sidebarDropRow = row
		}
		return gdk.ActionMove
	}))
	c.trackWidgetSignal(list, target, target.ConnectLeave(c.clearChatDropIndicator))
	c.trackWidgetSignal(list, target, target.ConnectDrop(func(value *glib.Value, _ float64, y float64) bool {
		raw, ok := value.GoValue().(string)
		if !ok {
			return false
		}
		payload, ok := decodeChatDrag(raw)
		if !ok || c.store == nil {
			return false
		}
		_, item, after := c.chatDropItem(list, archived, y)
		if err := c.applyChatDrop(payload, item, after); err != nil {
			c.showFolderError(err)
			return false
		}
		c.clearChatDropIndicator()
		// Complete GTK's drag before replacing its source widget.
		c.queueSidebarReload()
		return true
	}))
	list.AddController(target)
}

func (c *MainController) applyChatDrop(payload chatDragPayload, item chatRowItem, after bool) error {
	ctx := context.Background()
	if payload.FolderID != "" {
		before := item.folderID
		if before != "" && (after || !item.isFolderHeader) {
			before = c.nextFolderID(before)
		}
		return c.store.MoveChatFolder(ctx, payload.FolderID, before)
	}
	before := ""
	if item.folderID != "" && !item.isFolderHeader {
		before = item.chat.JID
		if after {
			before = ""
			for _, f := range c.chatFolders {
				if f.ID != item.folderID {
					continue
				}
				for i, jid := range f.ChatJIDs {
					if jid == item.chat.JID && i+1 < len(f.ChatJIDs) {
						before = f.ChatJIDs[i+1]
						break
					}
				}
			}
		}
	}
	return c.store.MoveFolderChats(ctx, payload.ChatJIDs, item.folderID, before)
}

func (c *MainController) nextFolderID(id string) string {
	for i, f := range c.chatFolders {
		if f.ID == id && i+1 < len(c.chatFolders) {
			return c.chatFolders[i+1].ID
		}
	}
	return ""
}

func (c *MainController) queueSidebarReload() {
	if c.sidebarRefreshQueued {
		return
	}
	c.sidebarRefreshQueued = true
	glib.IdleAdd(func() {
		c.sidebarRefreshQueued = false
		if c.sidebarDrag == nil {
			c.reloadFolderLists()
		}
	})
}
