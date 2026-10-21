package media

import (
	"context"
	"strings"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/store"
)

// LoadMediaPreview sets the paintable on picture, loading synchronously if cached
// or asynchronously downloading the media in the background.
func LoadMediaPreview(client Downloader, msg store.StoredMessage, picture *gtk.Picture, onLoaded func(tex *gdk.Texture)) {
	if tex := GetMediaThumbnailTexture(msg); tex != nil {
		picture.SetPaintable(tex)
		if onLoaded != nil {
			onLoaded(tex)
		}
		return
	}

	waMsg := ParseE2EMessage(msg.Raw)
	if waMsg == nil || (waMsg.GetImageMessage() == nil && waMsg.GetVideoMessage() == nil && waMsg.GetDocumentMessage() == nil && waMsg.GetStickerMessage() == nil) {
		return
	}

	key := msg.ChatJID + "_" + msg.ID
	go func() {
		path, err := DownloadMedia(context.Background(), client, msg)
		if err != nil {
			return
		}
		var tex *gdk.Texture
		lower := strings.ToLower(path)
		if strings.HasSuffix(lower, ".jpg") || strings.HasSuffix(lower, ".jpeg") || strings.HasSuffix(lower, ".png") || strings.HasSuffix(lower, ".webp") {
			tex, _ = gdk.NewTextureFromFilename(path)
		} else if strings.HasSuffix(lower, ".pdf") {
			tex = GetPDFThumbnailTexture(path)
		} else if strings.HasSuffix(lower, ".mp4") || strings.HasSuffix(lower, ".mov") || strings.HasSuffix(lower, ".mkv") || strings.HasSuffix(lower, ".webm") {
			tex = GetVideoThumbnailTexture(path)
		}

		if tex != nil {
			mediaTextureMu.Lock()
			mediaTextureCache[key] = tex
			mediaTextureMu.Unlock()
			glib.IdleAdd(func() {
				picture.SetPaintable(tex)
				if onLoaded != nil {
					onLoaded(tex)
				}
			})
		}
	}()
}
