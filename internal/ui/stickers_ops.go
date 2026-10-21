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
	"postbetasoftware.com/whatsdown/internal/media"
	"postbetasoftware.com/whatsdown/internal/store"
)

// isStickerSaved checks if a sticker message has already been saved.
func (c *MainController) isStickerSaved(msg store.StoredMessage) bool {
	if c.savedStickerHashes != nil {
		if c.savedStickerHashes[msg.ID] {
			return true
		}
		localPath := media.GetLocalMediaPath(msg)
		if fi, err := os.Stat(localPath); err == nil && fi.Size() > 0 {
			if data, err := os.ReadFile(localPath); err == nil {
				hash := fmt.Sprintf("%x", sha256.Sum256(data))
				if c.savedStickerHashes[hash] {
					return true
				}
			}
		}
	}
	return false
}

// saveStickerFromMessage saves a sticker from a chat message into the user's collection.
func (c *MainController) saveStickerFromMessage(msg store.StoredMessage) {
	go func() {
		localPath, err := media.DownloadMedia(context.Background(), c.client, msg)
		if err != nil {
			log.Printf("saving sticker: download failed: %v", err)
			return
		}
		data, err := os.ReadFile(localPath)
		if err != nil {
			log.Printf("saving sticker: reading file failed: %v", err)
			return
		}
		hash := fmt.Sprintf("%x", sha256.Sum256(data))
		destDir := media.SavedStickersDir()
		destFile := filepath.Join(destDir, hash+".webp")
		_ = os.WriteFile(destFile, data, 0o600)

		sticker := store.SavedSticker{
			ID:       hash,
			FilePath: destFile,
			FileHash: hash,
			Width:    512,
			Height:   512,
			SavedAt:  time.Now(),
		}
		if waMsg := media.ParseE2EMessage(msg.Raw); waMsg != nil && waMsg.GetStickerMessage() != nil {
			if w := int(waMsg.GetStickerMessage().GetWidth()); w > 0 {
				sticker.Width = w
			}
			if h := int(waMsg.GetStickerMessage().GetHeight()); h > 0 {
				sticker.Height = h
			}
		}
		if c.store != nil {
			_ = c.store.SaveSticker(context.Background(), sticker)
		}
		glib.IdleAdd(func() {
			c.loadSavedStickersUI()
		})
	}()
}
