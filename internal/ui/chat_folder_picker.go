package ui

import (
	"context"
	"fmt"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/store"
)

// Use the same membership operation as dragging, including multi-chat selection.
func (c *MainController) showAddToFolderDialog(jids []string) *adw.Dialog {
	if c.store == nil || len(jids) == 0 {
		return nil
	}
	folders, err := c.store.ListChatFolders(context.Background())
	if err != nil {
		c.showFolderError(err)
		return nil
	}
	dialog := adw.NewDialog()
	dialog.SetTitle("Add to Folder")
	dialog.SetContentWidth(360)
	dialog.SetContentHeight(400)
	toolbar := adw.NewToolbarView()
	toolbar.AddTopBar(adw.NewHeaderBar())
	scroll := gtk.NewScrolledWindow()
	scroll.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)
	box := gtk.NewBox(gtk.OrientationVertical, 12)
	box.SetMarginTop(12)
	box.SetMarginBottom(12)
	box.SetMarginStart(12)
	box.SetMarginEnd(12)
	list := gtk.NewListBox()
	list.SetSelectionMode(gtk.SelectionNone)
	list.AddCSSClass("boxed-list")
	errorLabel := gtk.NewLabel("")
	errorLabel.SetWrap(true)
	errorLabel.AddCSSClass("error")
	errorLabel.SetVisible(false)
	for _, folder := range folders {
		row := adw.NewActionRow()
		row.SetUseMarkup(false)
		row.SetTitle(folder.Name)
		row.SetSubtitle(fmt.Sprintf("%d chats", len(folder.ChatJIDs)))
		row.AddPrefix(gtk.NewImageFromIconName("folder-symbolic"))
		row.SetActivatable(true)
		if folderContainsChats(folder, jids) {
			row.SetSubtitle("Already in this folder")
			row.SetSensitive(false)
		}
		c.trackWidgetSignal(row, row, row.ConnectActivated(func() {
			if err := c.store.MoveFolderChats(context.Background(), jids, folder.ID, ""); err != nil {
				errorLabel.SetText(err.Error())
				errorLabel.SetVisible(true)
				return
			}
			dialog.Close()
			c.reloadFolderLists()
		}))
		list.Append(row)
	}
	create := adw.NewActionRow()
	create.SetTitle("New Folder…")
	create.AddPrefix(gtk.NewImageFromIconName("folder-new-symbolic"))
	create.SetActivatable(true)
	c.trackWidgetSignal(create, create, create.ConnectActivated(func() { dialog.Close(); c.showChatFolderDialog(nil, jids) }))
	list.Append(create)
	box.Append(list)
	box.Append(errorLabel)
	scroll.SetChild(box)
	toolbar.SetContent(scroll)
	dialog.SetChild(toolbar)
	c.trackWidgetSignal(dialog, dialog, dialog.ConnectClosed(func() { c.releaseMessageWidget(dialog) }))
	dialog.Present(c.window)
	return dialog
}
func folderContainsChats(folder store.ChatFolder, jids []string) bool {
	members := make(map[string]bool, len(folder.ChatJIDs))
	for _, jid := range folder.ChatJIDs {
		members[jid] = true
	}
	for _, jid := range jids {
		if !members[jid] {
			return false
		}
	}
	return len(jids) > 0
}
