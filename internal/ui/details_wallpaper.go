package ui

import (
	"context"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/store"
)

func (c *MainController) buildDetailsWallpaperGroup(dialog *adw.Dialog, chat store.Chat) *adw.PreferencesGroup {
	group := adw.NewPreferencesGroup()
	group.SetTitle("Chat Background")
	group.SetDescription("Customize wallpaper for this conversation")

	wp, _ := c.store.GetChatWallpaper(context.Background(), chat.JID)

	previewBox, pw := c.buildWallpaperPreview()
	pw.update(wp)
	previewRow := adw.NewActionRow()
	previewRow.SetTitle("Preview")
	previewRow.SetActivatable(false)
	previewRow.SetChild(previewBox)
	group.Add(previewRow)

	typeRow := adw.NewComboRow()
	typeRow.SetTitle("Background Type")
	typeRow.SetModel(gtk.NewStringList([]string{"Default (None)", "Solid Color", "Custom Image"}))

	colorRow := adw.NewActionRow()
	colorRow.SetTitle("Color")
	colorBtn := gtk.NewColorDialogButton(gtk.NewColorDialog())
	colorBtn.SetVAlign(gtk.AlignCenter)
	colorRow.AddSuffix(colorBtn)

	imageRow := adw.NewActionRow()
	imageRow.SetTitle("Image")
	chooseImgBtn := gtk.NewButtonWithLabel("Choose Image…")
	chooseImgBtn.SetVAlign(gtk.AlignCenter)
	imageRow.AddSuffix(chooseImgBtn)

	c.initWallpaperUI(wp, typeRow, colorBtn, imageRow)

	typeRow.Connect("notify::selected", func() {
		c.handleWallpaperTypeChange(chat, typeRow, colorRow, imageRow, colorBtn, pw)
	})

	colorBtn.NotifyProperty("rgba", func() {
		c.handleWallpaperColorChange(chat, colorBtn, pw)
	})

	chooseImgBtn.ConnectClicked(func() {
		c.openWallpaperImageChooser(chat, imageRow, pw)
	})

	group.Add(typeRow)
	group.Add(colorRow)
	group.Add(imageRow)
	return group
}

func (c *MainController) handleWallpaperTypeChange(
	chat store.Chat,
	typeRow *adw.ComboRow,
	colorRow, imageRow *adw.ActionRow,
	colorBtn *gtk.ColorDialogButton,
	pw *wallpaperPreviewWidgets,
) {
	sel := typeRow.Selected()
	colorRow.SetVisible(sel == 1)
	imageRow.SetVisible(sel == 2)

	switch sel {
	case 0:
		_ = c.store.SetChatWallpaper(context.Background(), chat.JID, "none", "")
		c.applyChatWallpaper(chat.JID)
		c.LoadChats()
		c.LoadArchivedChats()
		pw.update(store.ChatWallpaper{ChatJID: chat.JID, Kind: "none"})
	case 1:
		colStr := "#1e1e1e"
		if rgba := colorBtn.RGBA(); rgba != nil {
			colStr = rgbaToHex(rgba)
		}
		_ = c.store.SetChatWallpaper(context.Background(), chat.JID, "color", colStr)
		c.applyChatWallpaper(chat.JID)
		c.LoadChats()
		c.LoadArchivedChats()
		pw.update(store.ChatWallpaper{ChatJID: chat.JID, Kind: "color", Value: colStr})
	case 2:
		c.openWallpaperImageChooser(chat, imageRow, pw)
	}
}
