package ui

import (
	"context"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/store"
)

func (c *MainController) handleAppShortcut(key uint, state gdk.ModifierType) bool {
	if c.window != nil && c.window.VisibleDialog() != nil {
		return false
	}
	if state&(gdk.AltMask|gdk.SuperMask) != 0 || state&gdk.ControlMask == 0 {
		return false
	}
	switch key {
	case gdk.KEY_Tab, gdk.KEY_ISO_Left_Tab:
		delta := 1
		if state&gdk.ShiftMask != 0 || key == gdk.KEY_ISO_Left_Tab {
			delta = -1
		}
		if chat, ok := c.adjacentVisibleChat(delta); ok {
			if c.sidebarStack != nil && c.sidebarStack.VisibleChildName() != "chats" && c.sidebarStack.VisibleChildName() != "archived" {
				c.sidebarStack.SetVisibleChildName("chats")
			}
			archived := c.sidebarStack != nil && c.sidebarStack.VisibleChildName() == "archived"
			rows, list := c.displayedChatRows, c.chatsList
			if archived {
				rows, list = c.displayedArchivedRows, c.archivedChatsList
			}
			if list != nil {
				for i, item := range rows {
					if item.chat.JID == chat.JID {
						c.selectOnlyChatRow(list.RowAtIndex(i), chat, archived)
						break
					}
				}
			}
			c.selectChat(chat)
		}
		return true
	case gdk.KEY_slash, gdk.KEY_question:
		c.showKeyboardShortcuts()
		return true
	case gdk.KEY_g, gdk.KEY_G:
		if c.currentChat != nil && c.window != nil {
			c.openFileChooserForMedia(false)
		}
		return true
	}
	if !c.composerFocused() {
		return false
	}
	var style string
	switch key {
	case gdk.KEY_b, gdk.KEY_B:
		style = "bold"
	case gdk.KEY_i, gdk.KEY_I:
		style = "italic"
	case gdk.KEY_s, gdk.KEY_S:
		style = "strike"
	case gdk.KEY_q, gdk.KEY_Q:
		style = "quote"
	case gdk.KEY_m, gdk.KEY_M:
		style = "monospace"
	default:
		return false
	}
	c.formatComposerSelection(style)
	return true
}

func (c *MainController) adjacentVisibleChat(delta int) (store.Chat, bool) {
	rows, list := c.displayedChatRows, c.chatsList
	if c.sidebarStack != nil && c.sidebarStack.VisibleChildName() == "archived" {
		rows, list = c.displayedArchivedRows, c.archivedChatsList
	}
	var chats []store.Chat
	current := -1
	for i, item := range rows {
		if item.isFolderHeader || item.isLooseDropZone || item.isCommunityHeader || item.chat.JID == "" {
			continue
		}
		if list != nil {
			row := list.RowAtIndex(i)
			if row == nil || !row.Visible() {
				continue
			}
		}
		if c.currentChat != nil && c.isSameChat(c.currentChat.JID, item.chat.JID) {
			current = len(chats)
		}
		chats = append(chats, item.chat)
	}
	if len(chats) == 0 {
		return store.Chat{}, false
	}
	if current < 0 {
		if delta < 0 {
			return chats[len(chats)-1], true
		}
		return chats[0], true
	}
	return chats[(current+delta+len(chats))%len(chats)], true
}

func (c *MainController) editLastSentMessage() bool {
	if c.currentChat == nil || c.messageEntry == nil || c.getMessageText() != "" || c.editingMessage != nil {
		return false
	}
	if c.store != nil {
		msg, err := c.store.LatestOutgoingText(context.Background(), c.currentChat.JID)
		if err != nil {
			c.messageActionError(err)
			return true
		}
		if msg != nil {
			c.startEditing(*msg)
			return true
		}
		return false
	}
	for i := len(c.currentMessages) - 1; i >= 0; i-- {
		msg := c.currentMessages[i]
		if msg.FromMe && msg.Kind == store.MessageKindText {
			c.startEditing(msg)
			return true
		}
	}
	return false
}

func (c *MainController) showKeyboardShortcuts() *adw.Dialog {
	dialog := adw.NewDialog()
	dialog.SetTitle("Keyboard Shortcuts")
	dialog.SetContentWidth(520)
	dialog.SetContentHeight(620)
	toolbar := adw.NewToolbarView()
	toolbar.AddTopBar(adw.NewHeaderBar())
	scroll := gtk.NewScrolledWindow()
	scroll.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)
	box := gtk.NewBox(gtk.OrientationVertical, 24)
	box.SetMarginTop(18)
	box.SetMarginBottom(18)
	box.SetMarginStart(18)
	box.SetMarginEnd(18)
	add := func(title, description string, shortcuts [][2]string) {
		group := adw.NewPreferencesGroup()
		group.SetTitle(title)
		if description != "" {
			group.SetDescription(description)
		}
		list := gtk.NewListBox()
		list.SetSelectionMode(gtk.SelectionNone)
		list.AddCSSClass("boxed-list")
		for _, shortcut := range shortcuts {
			row := adw.NewActionRow()
			row.SetTitle(shortcut[1])
			row.SetUseMarkup(false)
			row.AddSuffix(gtk.NewShortcutLabel(shortcut[0]))
			list.Append(row)
		}
		group.Add(list)
		box.Append(group)
	}
	add("Chats", "", [][2]string{{"<Control>Tab", "Next chat"}, {"<Control><Shift>Tab", "Previous chat"}})
	add("Selected Text", "Select text in the message box before formatting it.", [][2]string{{"<Control>b", "Bold"}, {"<Control>i", "Italic"}, {"<Control>s", "Strikethrough"}, {"<Control>q", "Quote selection"}, {"<Control>m", "Monospace"}})
	add("Messages", "", [][2]string{{"Up", "Edit last sent text message (empty composer)"}, {"<Control>g", "Send a file"}})
	add("Help", "", [][2]string{{"<Control>slash", "Show keyboard shortcuts"}, {"Escape", "Close viewer or dialog"}})
	scroll.SetChild(box)
	toolbar.SetContent(scroll)
	dialog.SetChild(toolbar)
	dialog.Present(c.window)
	return dialog
}
