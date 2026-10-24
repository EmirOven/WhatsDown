package ui

import (
	"fmt"
	"strings"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotk4/pkg/pango"
	"postbetasoftware.com/whatsdown/internal/avatar"
	"postbetasoftware.com/whatsdown/internal/store"
)

func (c *MainController) chatFolderID(jid string) string {
	for _, f := range c.chatFolders {
		for _, member := range f.ChatJIDs {
			if member == jid {
				return f.ID
			}
		}
	}
	return ""
}

func chatMatches(chat store.Chat, query string) bool {
	return query == "" || strings.Contains(strings.ToLower(chat.Name), query) || strings.Contains(strings.ToLower(chat.LastMessageText), query)
}

// Extract memberships before WhatsApp community grouping. Manual folders are
// flat; their children follow stored order rather than last-message timestamps.
func folderChatSections(chats []store.Chat, folders []store.ChatFolder) (map[string][]store.Chat, []store.Chat) {
	byJID := make(map[string]store.Chat, len(chats))
	for _, chat := range chats {
		byJID[chat.JID] = chat
	}
	grouped := make(map[string][]store.Chat)
	assigned := make(map[string]bool)
	for _, f := range folders {
		for _, jid := range f.ChatJIDs {
			if chat, ok := byJID[jid]; ok && !assigned[jid] {
				grouped[f.ID] = append(grouped[f.ID], chat)
				assigned[jid] = true
			}
		}
	}
	var loose []store.Chat
	for _, chat := range chats {
		if !assigned[chat.JID] {
			// A community header assigned to a manual folder must not be
			// synthesized a second time above its remaining loose children.
			if assigned[chat.ParentJID] {
				chat.ParentJID = ""
			}
			loose = append(loose, chat)
		}
	}
	return grouped, loose
}

func (c *MainController) appendChatFolderRows(list *gtk.ListBox, chats []store.Chat, query string, archived bool, rows *[]chatRowItem) []store.Chat {
	grouped, loose := folderChatSections(chats, c.chatFolders)
	for _, folder := range c.chatFolders {
		children := grouped[folder.ID]
		if archived && len(children) == 0 {
			continue
		}
		nameMatches := query == "" || strings.Contains(strings.ToLower(folder.Name), query)
		matching := nameMatches
		for _, chat := range children {
			matching = matching || chatMatches(chat, query)
		}
		if !matching {
			continue
		}
		// Search reveals matching children without overwriting the collapse preference.
		expanded := !folder.Collapsed || query != ""
		header := c.createChatFolderRow(folder, children, expanded)
		list.Append(header)
		*rows = append(*rows, chatRowItem{isFolderHeader: true, folderID: folder.ID})
		for _, chat := range children {
			if !nameMatches && !chatMatches(chat, query) {
				continue
			}
			row := c.createChatRow(chat, archived)
			row.AddCSSClass("folder-chat-row")
			row.SetVisible(expanded)
			list.Append(row)
			*rows = append(*rows, chatRowItem{chat: chat, folderID: folder.ID})
		}
	}
	if len(*rows) > 0 {
		zone := gtk.NewListBoxRow()
		zone.SetSelectable(false)
		zone.SetActivatable(false)
		zone.AddCSSClass("chat-loose-heading")
		label := gtk.NewLabel("Other chats")
		label.SetXAlign(0)
		zone.SetChild(label)
		zone.SetTooltipText("Drop chats here to remove them from a folder")
		list.Append(zone)
		*rows = append(*rows, chatRowItem{isLooseDropZone: true})
	}
	return loose
}

func (c *MainController) createChatFolderRow(folder store.ChatFolder, children []store.Chat, expanded bool) *gtk.ListBoxRow {
	row := gtk.NewListBoxRow()
	row.AddCSSClass("chat-folder-row")
	row.SetSelectable(false)
	box := gtk.NewBox(gtk.OrientationHorizontal, 10)
	bar := gtk.NewBox(gtk.OrientationVertical, 0)
	bar.AddCSSClass("chat-row-color-bar")
	bar.SetVExpand(true)
	provider := gtk.NewCSSProvider()
	provider.LoadFromData(fmt.Sprintf(".chat-row-color-bar { background-color: %s; }", folder.Color))
	bar.StyleContext().AddProvider(provider, gtk.STYLE_PROVIDER_PRIORITY_APPLICATION+20)
	box.Append(bar)
	box.Append(c.createFolderAvatar(folder, children))
	label := gtk.NewLabel(folder.Name)
	label.SetEllipsize(pango.EllipsizeEnd)
	label.AddCSSClass("heading")
	label.SetXAlign(0)
	label.SetHExpand(true)
	box.Append(label)
	badge := gtk.NewLabel(fmt.Sprint(len(children)))
	badge.AddCSSClass("dim-label")
	box.Append(badge)
	arrow := "pan-end-symbolic"
	if expanded {
		arrow = "pan-down-symbolic"
	}
	box.Append(gtk.NewImageFromIconName(arrow))
	row.SetChild(box)
	c.attachFolderContextMenu(row, folder)
	return row
}

func (c *MainController) reloadFolderLists() {
	c.loadChats()
	c.loadArchivedChats()
}

func (c *MainController) toggleChatFolder(id string) {
	if c.store == nil {
		return
	}
	for i := range c.chatFolders {
		if c.chatFolders[i].ID != id {
			continue
		}
		c.chatFolders[i].Collapsed = !c.chatFolders[i].Collapsed
		collapsed := c.chatFolders[i].Collapsed
		c.setFolderRowsExpanded(c.chatsList, c.displayedChatRows, id, !collapsed || c.chatsSearchFilter != "")
		c.setFolderRowsExpanded(c.archivedChatsList, c.displayedArchivedRows, id, !collapsed || c.archivedSearchFilter != "")
		c.persistFolderCollapsed(id, collapsed)
		return
	}
}

// Keep the folder avatar at chat-avatar size, with overlapping member portraits.
func (c *MainController) createFolderAvatar(folder store.ChatFolder, children []store.Chat) *gtk.Fixed {
	cluster := gtk.NewFixed()
	cluster.SetSizeRequest(40, 40)
	cluster.SetVAlign(gtk.AlignCenter)
	cluster.SetTooltipText(folder.Name)
	if len(children) == 0 {
		icon := gtk.NewImageFromIconName("folder-symbolic")
		icon.SetPixelSize(32)
		cluster.Put(icon, 4, 4)
		return cluster
	}
	size := 24
	if len(children) == 1 {
		size = 40
	}
	// A compact cluster shows up to four portraits; the count beside it includes every chat.
	positions := [][2]float64{{0, 0}, {16, 0}, {0, 16}, {16, 16}}
	if len(children) == 2 {
		positions = [][2]float64{{0, 0}, {16, 16}}
	}
	if len(children) == 3 {
		positions = [][2]float64{{8, 0}, {0, 16}, {16, 16}}
	}
	for i, chat := range children {
		if i == len(positions) {
			break
		}
		name := formatDisplayName(chat)
		av := adw.NewAvatar(size, name, true)
		av.AddCSSClass("folder-member-avatar")
		avatar.LoadAvatar(c.client, chat.JID, name, av)
		cluster.Put(av, positions[i][0], positions[i][1])
	}
	return cluster
}
