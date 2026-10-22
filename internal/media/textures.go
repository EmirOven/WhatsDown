package media

import (
	"os"
	"strings"

	coreglib "github.com/diamondburned/gotk4/pkg/core/glib"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gdkpixbuf/v2"
	"postbetasoftware.com/whatsdown/internal/cache"
	"postbetasoftware.com/whatsdown/internal/store"
)

// Decoded textures live outside the Go heap; bound their estimated RGBA size.
var mediaTextureCache = cache.NewLRU[*gdk.Texture](64<<20, 256)

// Keep previews small even when a chat contains many high-resolution photos.
// Opening the attachment still uses the original file.
func loadPreviewTexture(path string) (*gdk.Texture, error) {
	return LoadSizedPreviewTexture(path, 640)
}

// LoadSizedPreviewTexture bounds decoded image dimensions without upscaling.
func LoadSizedPreviewTexture(path string, maxSize int) (*gdk.Texture, error) {
	// AtScale also enlarges small images. Preserve their original dimensions;
	// a 64px thumbnail must not become a 640px native allocation.
	width, height, _ := gdkpixbuf.PixbufGetFileInfo(path)
	targetWidth, targetHeight := maxSize, maxSize
	if width > 0 && height > 0 && width <= maxSize && height <= maxSize {
		targetWidth, targetHeight = width, height
	}
	pixbuf, err := gdkpixbuf.NewPixbufFromFileAtScale(path, targetWidth, targetHeight, true)
	if err != nil {
		return nil, err
	}
	// The texture acquires its own native reference. Release this temporary
	// wrapper now instead of waiting for Go GC to notice native allocations.
	defer coreglib.Destroy(pixbuf)
	return gdk.NewTextureForPixbuf(pixbuf), nil
}

func cacheMediaTexture(key string, tex *gdk.Texture) {
	if tex != nil {
		mediaTextureCache.Put(key, tex, int64(tex.Width())*int64(tex.Height())*4)
	}
}

// GetMediaThumbnailTexture returns a GdkTexture for a message's thumbnail if available.
func GetMediaThumbnailTexture(msg store.StoredMessage) *gdk.Texture {
	key := msg.ChatJID + "_" + msg.ID
	if tex, ok := mediaTextureCache.Get(key); ok && tex != nil {
		return tex
	}

	localPath := GetLocalMediaPath(msg)
	if localPath != "" {
		if fi, err := os.Stat(localPath); err == nil && fi.Size() > 0 {
			lower := strings.ToLower(localPath)
			if strings.HasSuffix(lower, ".jpg") || strings.HasSuffix(lower, ".jpeg") || strings.HasSuffix(lower, ".png") || strings.HasSuffix(lower, ".webp") {
				tex, err := loadPreviewTexture(localPath)
				if err == nil && tex != nil {
					cacheMediaTexture(key, tex)
					return tex
				}
			} else if strings.HasSuffix(lower, ".pdf") {
				if tex := GetPDFThumbnailTexture(localPath); tex != nil {
					cacheMediaTexture(key, tex)
					return tex
				}
			} else if strings.HasSuffix(lower, ".mp4") || strings.HasSuffix(lower, ".mov") || strings.HasSuffix(lower, ".mkv") || strings.HasSuffix(lower, ".webm") {
				if tex := GetVideoThumbnailTexture(localPath); tex != nil {
					cacheMediaTexture(key, tex)
					return tex
				}
			}
		}
	}

	thumbPath := GetThumbCachePath(msg)
	if fi, err := os.Stat(thumbPath); err == nil && fi.Size() > 0 {
		tex, err := loadPreviewTexture(thumbPath)
		if err == nil && tex != nil {
			cacheMediaTexture(key, tex)
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
			tex, err := loadPreviewTexture(thumbPath)
			if err == nil && tex != nil {
				cacheMediaTexture(key, tex)
				return tex
			}
		}
	}

	return nil
}
