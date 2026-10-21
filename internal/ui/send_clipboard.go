package ui

import (
	"context"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
)

// handleClipboardMediaPaste checks if the clipboard contains an image or file URI.
func (c *MainController) handleClipboardMediaPaste() bool {
	if c.currentChat == nil || c.window == nil {
		return false
	}
	clipboard := c.window.Clipboard()
	if clipboard == nil {
		if disp := gdk.DisplayGetDefault(); disp != nil {
			clipboard = disp.Clipboard()
		}
	}
	if clipboard == nil {
		return false
	}
	formats := clipboard.Formats()
	if formats == nil {
		return false
	}

	hasImage, hasURIList := c.inspectClipboardFormats(formats)
	if !hasImage && !hasURIList {
		return false
	}

	if hasImage {
		c.readClipboardImage(clipboard)
		return true
	}

	clipboard.ReadTextAsync(context.Background(), func(res gio.AsyncResulter) {
		text, err := clipboard.ReadTextFinish(res)
		if err != nil || text == "" {
			return
		}
		if targetPath, ok := parseFilePathFromText(text); ok {
			glib.IdleAdd(func() {
				c.showSendAttachmentDialog(targetPath)
			})
		}
	})
	return true
}

func (c *MainController) inspectClipboardFormats(formats *gdk.ContentFormats) (bool, bool) {
	hasImage, hasURIList := false, false
	for _, gtype := range formats.GTypes() {
		if gtype.IsA(gdk.GTypeTexture) || gtype.IsA(gdk.GTypePaintable) {
			hasImage = true
			break
		}
		if gtype.IsA(gdk.GTypeFileList) {
			hasURIList = true
		}
	}
	for _, mime := range formats.MIMETypes() {
		if stringsHasPrefix(mime, "image/") {
			hasImage = true
		}
		if mime == "text/uri-list" || mime == "x-special/gnome-copied-files" ||
			mime == "application/vnd.portal.filetransfer" || mime == "application/vnd.portal.files" {
			hasURIList = true
		}
	}
	return hasImage, hasURIList
}

func stringsHasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[0:len(prefix)] == prefix
}
