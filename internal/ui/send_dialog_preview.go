package ui

import (
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/media"
)

func buildImageSendPreview(filePath string) gtk.Widgetter {
	pic := gtk.NewPicture()
	pic.SetKeepAspectRatio(true)
	pic.SetCanShrink(true)
	pic.SetSizeRequest(360, 260)
	pic.AddCSSClass("chat-bubble-picture")
	if tex, err := gdk.NewTextureFromFilename(filePath); err == nil && tex != nil {
		pic.SetPaintable(tex)
	}
	return pic
}

func buildVideoSendPreview(filePath string) gtk.Widgetter {
	overlay := gtk.NewOverlay()
	pic := gtk.NewPicture()
	pic.SetKeepAspectRatio(true)
	pic.SetCanShrink(true)
	pic.SetSizeRequest(360, 260)
	pic.AddCSSClass("chat-bubble-picture")
	if tex := media.GetVideoThumbnailTexture(filePath); tex != nil {
		pic.SetPaintable(tex)
	}
	overlay.SetChild(pic)

	playBtn := gtk.NewButtonFromIconName("media-playback-start-symbolic")
	addCSSClasses(playBtn, "circular", "suggested-action")
	playBtn.SetHAlign(gtk.AlignCenter)
	playBtn.SetVAlign(gtk.AlignCenter)
	playBtn.SetTooltipText("Preview Video")
	playBtn.ConnectClicked(func() {
		_ = media.OpenFile(filePath)
	})
	overlay.AddOverlay(playBtn)
	return overlay
}

func buildAudioSendPreview(filePath string) gtk.Widgetter {
	cardBox := gtk.NewBox(gtk.OrientationVertical, 12)
	cardBox.SetHAlign(gtk.AlignCenter)

	icon := gtk.NewImageFromIconName("folder-music-symbolic")
	icon.SetPixelSize(64)
	cardBox.Append(icon)

	playBtn := gtk.NewButton()
	addCSSClasses(playBtn, "pill", "suggested-action")
	playBtn.SetHAlign(gtk.AlignCenter)

	btnBox := gtk.NewBox(gtk.OrientationHorizontal, 6)
	btnIcon := gtk.NewImageFromIconName("media-playback-start-symbolic")
	btnLbl := gtk.NewLabel("Play Audio Preview")
	btnBox.Append(btnIcon)
	btnBox.Append(btnLbl)
	playBtn.SetChild(btnBox)

	isPlaying := false
	playBtn.ConnectClicked(func() {
		if isPlaying {
			media.StopCurrentAudio()
			return
		}
		media.PlayAudioPreview(filePath, func(playing bool) {
			isPlaying = playing
			if playing {
				btnIcon.SetFromIconName("media-playback-pause-symbolic")
				btnLbl.SetText("Pause Preview")
			} else {
				btnIcon.SetFromIconName("media-playback-start-symbolic")
				btnLbl.SetText("Play Audio Preview")
			}
		})
	})
	cardBox.Append(playBtn)
	return cardBox
}
