package media

import (
	"os"
	"strings"
	"sync"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"postbetasoftware.com/whatsdown/internal/store"
)

var (
	mediaTextureMu    sync.RWMutex
	mediaTextureCache = make(map[string]*gdk.Texture)
)

// GetMediaThumbnailTexture returns a GdkTexture for a message's thumbnail if available.
func GetMediaThumbnailTexture(msg store.StoredMessage) *gdk.Texture {
	key := msg.ChatJID + "_" + msg.ID
	mediaTextureMu.RLock()
	if tex, ok := mediaTextureCache[key]; ok && tex != nil {
		mediaTextureMu.RUnlock()
		return tex
	}
	mediaTextureMu.RUnlock()

	localPath := GetLocalMediaPath(msg)
	if localPath != "" {
		if fi, err := os.Stat(localPath); err == nil && fi.Size() > 0 {
			lower := strings.ToLower(localPath)
			if strings.HasSuffix(lower, ".jpg") || strings.HasSuffix(lower, ".jpeg") || strings.HasSuffix(lower, ".png") || strings.HasSuffix(lower, ".webp") {
				tex, err := gdk.NewTextureFromFilename(localPath)
				if err == nil && tex != nil {
					mediaTextureMu.Lock()
					mediaTextureCache[key] = tex
					mediaTextureMu.Unlock()
					return tex
				}
			} else if strings.HasSuffix(lower, ".pdf") {
				if tex := GetPDFThumbnailTexture(localPath); tex != nil {
					mediaTextureMu.Lock()
					mediaTextureCache[key] = tex
					mediaTextureMu.Unlock()
					return tex
				}
			} else if strings.HasSuffix(lower, ".mp4") || strings.HasSuffix(lower, ".mov") || strings.HasSuffix(lower, ".mkv") || strings.HasSuffix(lower, ".webm") {
				if tex := GetVideoThumbnailTexture(localPath); tex != nil {
					mediaTextureMu.Lock()
					mediaTextureCache[key] = tex
					mediaTextureMu.Unlock()
					return tex
				}
			}
		}
	}

	thumbPath := GetThumbCachePath(msg)
	if fi, err := os.Stat(thumbPath); err == nil && fi.Size() > 0 {
		tex, err := gdk.NewTextureFromFilename(thumbPath)
		if err == nil && tex != nil {
			mediaTextureMu.Lock()
			mediaTextureCache[key] = tex
			mediaTextureMu.Unlock()
			return tex
		}
	}

	waMsg := ParseE2EMessage(msg.Raw)
	if waMsg != nil {
		var thumbBytes []byte
		switch {
		case waMsg.GetImageMessage() != nil:
			thumbBytes = waMsg.GetImageMessage().GetJPEGThumbnail()
		case waMsg.GetVideoMessage() != nil:
			thumbBytes = waMsg.GetVideoMessage().GetJPEGThumbnail()
		case waMsg.GetDocumentMessage() != nil:
			thumbBytes = waMsg.GetDocumentMessage().GetJPEGThumbnail()
		case waMsg.GetStickerMessage() != nil:
			thumbBytes = waMsg.GetStickerMessage().GetPngThumbnail()
		}
		if len(thumbBytes) > 0 {
			_ = os.WriteFile(thumbPath, thumbBytes, 0o600)
			tex, err := gdk.NewTextureFromFilename(thumbPath)
			if err == nil && tex != nil {
				mediaTextureMu.Lock()
				mediaTextureCache[key] = tex
				mediaTextureMu.Unlock()
				return tex
			}
		}
	}

	return nil
}
