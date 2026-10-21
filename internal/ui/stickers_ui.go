package ui

import (
	"context"
	"log"
	"os"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/store"
)

// loadSavedStickersUI populates the sticker popover flowbox.
func (c *MainController) LoadSavedStickersUI() {
	if c.store == nil || c.stickerFlowbox == nil {
		return
	}
	stickers, err := c.store.ListSavedStickers(context.Background())
	if err != nil {
		log.Printf("listing saved stickers: %v", err)
		return
	}
	c.savedStickers = stickers
	c.savedStickerHashes = make(map[string]bool)
	for _, s := range stickers {
		c.savedStickerHashes[s.ID] = true
		if s.FileHash != "" {
			c.savedStickerHashes[s.FileHash] = true
		}
	}
	for {
		ch := c.stickerFlowbox.FirstChild()
		if ch == nil {
			break
		}
		c.stickerFlowbox.Remove(ch)
	}
	if len(stickers) == 0 {
		if c.stickerEmptyBox != nil {
			c.stickerEmptyBox.SetVisible(true)
		}
		if c.stickerScrolledWindow != nil {
			c.stickerScrolledWindow.SetVisible(false)
		}
		return
	}
	if c.stickerEmptyBox != nil {
		c.stickerEmptyBox.SetVisible(false)
	}
	if c.stickerScrolledWindow != nil {
		c.stickerScrolledWindow.SetVisible(true)
	}
	for _, s := range stickers {
		c.appendStickerToFlowbox(s)
	}
}

func (c *MainController) loadSavedStickersUI() {
	c.LoadSavedStickersUI()
}

func (c *MainController) appendStickerToFlowbox(stk store.SavedSticker) {
	btn := gtk.NewButton()
	addCSSClasses(btn, "flat", "sticker-picker-item")
	btn.SetCursorFromName("pointer")
	btn.SetTooltipText("Click to send")

	pic := gtk.NewPicture()
	pic.SetKeepAspectRatio(true)
	pic.SetCanShrink(true)
	pic.SetSizeRequest(64, 64)

	if fi, err := os.Stat(stk.FilePath); err == nil && fi.Size() > 0 {
		if tex, err := gdk.NewTextureFromFilename(stk.FilePath); err == nil && tex != nil {
			pic.SetPaintable(tex)
		}
	}
	btn.SetChild(pic)

	btn.ConnectClicked(func() {
		if c.stickerPopover != nil {
			c.stickerPopover.Popdown()
		}
		c.sendSticker(stk.FilePath)
	})

	c.attachStickerPickerMenu(btn, stk)
	c.stickerFlowbox.Append(btn)
}
