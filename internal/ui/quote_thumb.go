package ui

import (
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"postbetasoftware.com/whatsdown/internal/media"
	"postbetasoftware.com/whatsdown/internal/store"
)

func getQuoteThumbnail(origMsg *store.StoredMessage, quotedRaw *waE2E.Message) *gtk.Picture {
	if origMsg != nil && (origMsg.Kind == store.MessageKindImage || origMsg.Kind == store.MessageKindVideo) {
		if tex := media.GetMediaThumbnailTexture(*origMsg); tex != nil {
			thumbPic := gtk.NewPicture()
			thumbPic.SetKeepAspectRatio(true)
			thumbPic.SetCanShrink(true)
			thumbPic.SetSizeRequest(36, 36)
			thumbPic.SetPaintable(tex)
			thumbPic.AddCSSClass("chat-bubble-quote-thumb")
			return thumbPic
		}
	} else if quotedRaw != nil {
		var jpegBytes []byte
		if img := quotedRaw.GetImageMessage(); img != nil && len(img.GetJPEGThumbnail()) > 0 {
			jpegBytes = img.GetJPEGThumbnail()
		} else if vid := quotedRaw.GetVideoMessage(); vid != nil && len(vid.GetJPEGThumbnail()) > 0 {
			jpegBytes = vid.GetJPEGThumbnail()
		}
		if len(jpegBytes) > 0 {
			if tex, err := gdk.NewTextureFromBytes(glib.NewBytes(jpegBytes)); err == nil && tex != nil {
				thumbPic := gtk.NewPicture()
				thumbPic.SetKeepAspectRatio(true)
				thumbPic.SetCanShrink(true)
				thumbPic.SetSizeRequest(36, 36)
				thumbPic.SetPaintable(tex)
				thumbPic.AddCSSClass("chat-bubble-quote-thumb")
				return thumbPic
			}
		}
	}
	return nil
}
