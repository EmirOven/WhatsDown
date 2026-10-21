package media

import (
	"fmt"
	"path/filepath"
	"strings"

	"postbetasoftware.com/whatsdown/internal/avatar"
	"postbetasoftware.com/whatsdown/internal/store"
)

func GetThumbCachePath(msg store.StoredMessage) string {
	ext := ".jpg"
	if waMsg := ParseE2EMessage(msg.Raw); waMsg != nil && waMsg.GetStickerMessage() != nil {
		ext = ".png"
	}
	filename := fmt.Sprintf("thumb_%s_%s%s", avatar.SanitizeJIDFilename(msg.ChatJID), avatar.SanitizeJIDFilename(msg.ID), ext)
	return filepath.Join(CacheDir(), filename)
}

func GetLocalMediaPath(msg store.StoredMessage) string {
	waMsg := ParseE2EMessage(msg.Raw)
	ext := ".bin"
	if waMsg != nil {
		switch {
		case waMsg.GetImageMessage() != nil:
			ext = ".jpg"
			if mt := waMsg.GetImageMessage().GetMimetype(); strings.Contains(mt, "png") {
				ext = ".png"
			}
		case waMsg.GetVideoMessage() != nil:
			ext = ".mp4"
		case waMsg.GetAudioMessage() != nil:
			ext = ".ogg"
			if mt := waMsg.GetAudioMessage().GetMimetype(); strings.Contains(mt, "mp4") || strings.Contains(mt, "m4a") {
				ext = ".m4a"
			}
		case waMsg.GetDocumentMessage() != nil:
			doc := waMsg.GetDocumentMessage()
			if doc.GetFileName() != "" {
				ext = filepath.Ext(doc.GetFileName())
				if ext == "" {
					ext = ".pdf"
				}
			}
		case waMsg.GetStickerMessage() != nil:
			ext = ".webp"
		}
	}
	filename := fmt.Sprintf("%s_%s%s", avatar.SanitizeJIDFilename(msg.ChatJID), avatar.SanitizeJIDFilename(msg.ID), ext)
	return filepath.Join(CacheDir(), filename)
}
