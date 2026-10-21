package ui

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"

	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"postbetasoftware.com/whatsdown/internal/media"
	"postbetasoftware.com/whatsdown/internal/store"
)

// removeSavedStickerByMessage deletes a sticker associated with a message.
func (c *MainController) removeSavedStickerByMessage(msg store.StoredMessage) {
	localPath := media.GetLocalMediaPath(msg)
	var hash string
	if data, err := os.ReadFile(localPath); err == nil {
		hash = fmt.Sprintf("%x", sha256.Sum256(data))
	}
	if hash != "" {
		c.removeSavedSticker(hash)
	}
	c.removeSavedSticker(msg.ID)
}

// removeSavedSticker deletes a saved sticker by ID or hash.
func (c *MainController) removeSavedSticker(id string) {
	if c.store != nil {
		_ = c.store.DeleteSavedSticker(context.Background(), id)
	}
	glib.IdleAdd(func() {
		c.loadSavedStickersUI()
	})
}
