package media

import (
	"context"
	"strings"
	"time"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/store"
)

var previewWorkers = make(chan struct{}, 2)

// LoadMediaPreview sets the paintable on picture, loading synchronously if cached
// or asynchronously downloading the media in the background.
func LoadMediaPreview(client Downloader, msg store.StoredMessage, picture *gtk.Picture, onLoaded func(tex *gdk.Texture)) {
	LoadMediaPreviewContext(context.Background(), client, msg, picture, onLoaded)
}

// LoadMediaPreviewContext cancels queued work when its chat is replaced.
func LoadMediaPreviewContext(ctx context.Context, client Downloader, msg store.StoredMessage, picture *gtk.Picture, onLoaded func(tex *gdk.Texture)) {
	if ctx == nil {
		ctx = context.Background()
	}
	if ctx.Err() != nil {
		return
	}
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

	chatContext := ctx
	key := msg.ChatJID + "_" + msg.ID
	go func() {
		select {
		case previewWorkers <- struct{}{}:
		case <-ctx.Done():
			return
		}
		defer func() { <-previewWorkers }()
		if ctx.Err() != nil {
			return
		}
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		path, err := DownloadMedia(ctx, client, msg)
		if err != nil {
			return
		}
		var tex *gdk.Texture
		lower := strings.ToLower(path)
		if strings.HasSuffix(lower, ".jpg") || strings.HasSuffix(lower, ".jpeg") || strings.HasSuffix(lower, ".png") || strings.HasSuffix(lower, ".webp") {
			tex, _ = loadPreviewTexture(path)
		} else if strings.HasSuffix(lower, ".pdf") {
			tex = GetPDFThumbnailTexture(path)
		} else if strings.HasSuffix(lower, ".mp4") || strings.HasSuffix(lower, ".mov") || strings.HasSuffix(lower, ".mkv") || strings.HasSuffix(lower, ".webm") {
			tex = GetVideoThumbnailTexture(path)
		}

		if tex != nil && ctx.Err() == nil {
			cacheMediaTexture(key, tex)
			glib.IdleAdd(func() {
				if chatContext.Err() != nil {
					return
				}
				picture.SetPaintable(tex)
				if onLoaded != nil {
					onLoaded(tex)
				}
			})
		}
	}()
}
