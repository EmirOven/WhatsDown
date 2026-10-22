package ui

import (
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/media"
	"postbetasoftware.com/whatsdown/internal/store"
)

func (c *MainController) buildStickerWidget(msg store.StoredMessage) *gtk.Picture {
	stickerW := 140
	stickerH := 140
	if waMsg := media.ParseE2EMessage(msg.Raw); waMsg != nil && waMsg.GetStickerMessage() != nil {
		w := int(waMsg.GetStickerMessage().GetWidth())
		h := int(waMsg.GetStickerMessage().GetHeight())
		if w > 0 && h > 0 {
			if w >= h {
				stickerH = int(float64(h) * 140.0 / float64(w))
				stickerW = 140
			} else {
				stickerW = int(float64(w) * 140.0 / float64(h))
				stickerH = 140
			}
			if stickerW < 60 {
				stickerW = 60
			}
			if stickerH < 60 {
				stickerH = 60
			}
		}
	}

	pic := gtk.NewPicture()
	pic.SetKeepAspectRatio(true)
	pic.SetCanShrink(true)
	addCSSClasses(pic, "chat-bubble-picture", "chat-bubble-sticker-picture")
	pic.SetSizeRequest(stickerW, stickerH)
	pic.SetCursorFromName("pointer")

	media.LoadMediaPreviewContext(c.messagePreviewContext, c.client, msg, pic, func(tex *gdk.Texture) {
		if tex != nil {
			glib.IdleAdd(func() {
				tw, th := tex.Width(), tex.Height()
				if tw > 0 && th > 0 {
					sw, sh := 140, 140
					if tw >= th {
						sh = int(float64(th) * 140.0 / float64(tw))
						sw = 140
					} else {
						sw = int(float64(tw) * 140.0 / float64(th))
						sh = 140
					}
					if sw < 60 {
						sw = 60
					}
					if sh < 60 {
						sh = 60
					}
					pic.SetSizeRequest(sw, sh)
					pic.QueueResize()
				}
				if c.autoScrollBottom {
					c.scrollToBottom()
				}
			})
		}
	})

	mCopy := msg
	c.attachStickerContextMenu(pic, mCopy)
	return pic
}
