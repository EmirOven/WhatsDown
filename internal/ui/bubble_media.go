package ui

import (
	"fmt"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/media"
	"postbetasoftware.com/whatsdown/internal/store"
)

func (c *MainController) buildImageVideoWidget(msg store.StoredMessage) *gtk.Overlay {
	maxImageWidth := c.getMaxImageWidth()
	imgW, imgH := 0, 0
	if waMsg := media.ParseE2EMessage(msg.Raw); waMsg != nil {
		if img := waMsg.GetImageMessage(); img != nil {
			imgW, imgH = int(img.GetWidth()), int(img.GetHeight())
		} else if vid := waMsg.GetVideoMessage(); vid != nil {
			imgW, imgH = int(vid.GetWidth()), int(vid.GetHeight())
		}
	}
	targetW, targetH := computeImageDisplaySize(imgW, imgH, maxImageWidth)

	pic := gtk.NewPicture()
	pic.SetKeepAspectRatio(true)
	pic.SetCanShrink(true)
	pic.AddCSSClass("chat-bubble-picture")
	pic.SetSizeRequest(targetW, targetH)

	media.LoadMediaPreviewContext(c.messagePreviewContext, c.client, msg, pic, func(tex *gdk.Texture) {
		if tex != nil {
			glib.IdleAdd(func() {
				tw, th := computeImageDisplaySize(tex.Width(), tex.Height(), c.getMaxImageWidth())
				pic.SetSizeRequest(tw, th)
				pic.QueueResize()
			})
		}
	})

	overlay := gtk.NewOverlay()
	overlay.SetChild(pic)

	if msg.Kind == store.MessageKindVideo {
		badge := gtk.NewLabel("▶ Video")
		if e2e := media.ParseE2EMessage(msg.Raw); e2e != nil && e2e.GetVideoMessage() != nil {
			if s := e2e.GetVideoMessage().GetSeconds(); s > 0 {
				badge.SetText(fmt.Sprintf("▶ %d:%02d", s/60, s%60))
			}
		}
		badge.AddCSSClass("media-video-badge")
		badge.SetHAlign(gtk.AlignStart)
		badge.SetVAlign(gtk.AlignEnd)
		badge.SetMarginStart(6)
		badge.SetMarginBottom(6)
		overlay.AddOverlay(badge)
	}

	c.attachMediaOverlayButton(overlay, msg)
	return overlay
}
