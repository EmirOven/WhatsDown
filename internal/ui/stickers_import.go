package ui

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/media"
	"postbetasoftware.com/whatsdown/internal/store"
)

// openFileChooserForSticker allows the user to import an image to save as a sticker.
func (c *MainController) openFileChooserForSticker() {
	dialog := gtk.NewFileChooserNative("Choose Image for Sticker", &c.window.Window, gtk.FileChooserActionOpen, "_Select", "_Cancel")

	filter := gtk.NewFileFilter()
	filter.SetName("Sticker Images (*.webp, *.png, *.jpg, *.jpeg)")
	filter.AddMIMEType("image/webp")
	filter.AddMIMEType("image/png")
	filter.AddMIMEType("image/jpeg")
	filter.AddPattern("*.webp")
	filter.AddPattern("*.WEBP")
	filter.AddPattern("*.png")
	filter.AddPattern("*.PNG")
	filter.AddPattern("*.jpg")
	filter.AddPattern("*.jpeg")
	filter.AddPattern("*.JPG")
	filter.AddPattern("*.JPEG")
	dialog.SetFilter(filter)

	dialog.ConnectResponse(func(responseId int) {
		if responseId == int(gtk.ResponseAccept) {
			if file := dialog.File(); file != nil {
				if path := file.Path(); path != "" {
					c.addNewStickerFromFile(path)
				}
			}
		}
		dialog.Destroy()
	})
	dialog.Show()
}

// addNewStickerFromFile converts and saves an external image into saved stickers.
func (c *MainController) addNewStickerFromFile(filePath string) {
	go func() {
		data, err := media.EnsureWebPSticker(filePath)
		if err != nil {
			log.Printf("adding sticker from file %s: %v", filePath, err)
			return
		}
		hash := fmt.Sprintf("%x", sha256.Sum256(data))
		destFile := filepath.Join(media.SavedStickersDir(), hash+".webp")
		_ = os.WriteFile(destFile, data, 0o600)

		sticker := store.SavedSticker{
			ID:       hash,
			FilePath: destFile,
			FileHash: hash,
			Width:    512,
			Height:   512,
			SavedAt:  time.Now(),
		}
		if c.store != nil {
			_ = c.store.SaveSticker(context.Background(), sticker)
		}
		glib.IdleAdd(func() {
			c.loadSavedStickersUI()
		})
	}()
}
