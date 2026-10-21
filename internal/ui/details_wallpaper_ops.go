package ui

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/store"
)

func (c *MainController) initWallpaperUI(wp store.ChatWallpaper, typeRow *adw.ComboRow, colorBtn *gtk.ColorDialogButton, imageRow *adw.ActionRow) {
	switch wp.Kind {
	case "color":
		typeRow.SetSelected(1)
		if wp.Value != "" {
			if rgba, ok := parseHexColor(wp.Value); ok {
				colorBtn.SetRGBA(&rgba)
			}
		}
		imageRow.SetVisible(false)
	case "image":
		typeRow.SetSelected(2)
		if wp.Value != "" {
			imageRow.SetSubtitle(filepath.Base(wp.Value))
		}
	default:
		typeRow.SetSelected(0)
		imageRow.SetVisible(false)
	}
}

func (c *MainController) handleWallpaperColorChange(chat store.Chat, colorBtn *gtk.ColorDialogButton, pw *wallpaperPreviewWidgets) {
	if rgba := colorBtn.RGBA(); rgba != nil {
		colStr := rgbaToHex(rgba)
		_ = c.store.SetChatWallpaper(context.Background(), chat.JID, "color", colStr)
		c.applyChatWallpaper(chat.JID)
		pw.update(store.ChatWallpaper{ChatJID: chat.JID, Kind: "color", Value: colStr})
	}
}

func (c *MainController) openWallpaperImageChooser(chat store.Chat, imageRow *adw.ActionRow, pw *wallpaperPreviewWidgets) {
	chooser := gtk.NewFileChooserNative("Choose Chat Background Image", &c.window.Window, gtk.FileChooserActionOpen, "_Select", "_Cancel")
	filter := gtk.NewFileFilter()
	filter.SetName("Images")
	filter.AddMIMEType("image/*")
	filter.AddPattern("*.png")
	filter.AddPattern("*.jpg")
	filter.AddPattern("*.jpeg")
	filter.AddPattern("*.webp")
	chooser.SetFilter(filter)

	chooser.ConnectResponse(func(resp int) {
		if resp == int(gtk.ResponseAccept) {
			if file := chooser.File(); file != nil {
				path := file.Path()
				if path != "" {
					_ = c.store.SetChatWallpaper(context.Background(), chat.JID, "image", path)
					glib.IdleAdd(func() {
						imageRow.SetSubtitle(filepath.Base(path))
						c.applyChatWallpaper(chat.JID)
						pw.update(store.ChatWallpaper{ChatJID: chat.JID, Kind: "image", Value: path})
					})
				}
			}
		}
	})
	chooser.Show()
}

func rgbaToHex(rgba *gdk.RGBA) string {
	r := int(rgba.Red() * 255)
	g := int(rgba.Green() * 255)
	b := int(rgba.Blue() * 255)
	a := int(rgba.Alpha() * 255)
	if a == 255 {
		return fmt.Sprintf("#%02x%02x%02x", r, g, b)
	}
	return fmt.Sprintf("rgba(%d, %d, %d, %.2f)", r, g, b, rgba.Alpha())
}
