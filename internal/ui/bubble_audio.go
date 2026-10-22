package ui

import (
	"context"
	"fmt"

	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/media"
	"postbetasoftware.com/whatsdown/internal/store"
)

func (c *MainController) buildAudioWidget(msg store.StoredMessage) *gtk.Box {
	actionBox := gtk.NewBox(gtk.OrientationHorizontal, 8)
	actionBox.AddCSSClass("chat-bubble-audio-box")

	playBtn := gtk.NewButtonFromIconName("media-playback-start-symbolic")
	addCSSClasses(playBtn, "circular", "suggested-action")
	playBtn.SetTooltipText("Play Preview")

	infoBox := gtk.NewBox(gtk.OrientationVertical, 2)
	infoBox.SetVAlign(gtk.AlignCenter)

	durTxt := "Voice Message"
	if e2e := media.ParseE2EMessage(msg.Raw); e2e != nil && e2e.GetAudioMessage() != nil {
		if s := e2e.GetAudioMessage().GetSeconds(); s > 0 {
			durTxt = fmt.Sprintf("Voice Note (%d:%02d)", s/60, s%60)
		}
	} else if msg.Kind == store.MessageKindAudio {
		durTxt = "Audio"
	}
	titleLbl := gtk.NewLabel(durTxt)
	titleLbl.SetXAlign(0)
	titleLbl.AddCSSClass("caption")

	progBar := gtk.NewProgressBar()
	progBar.SetFraction(0.0)
	progBar.SetSizeRequest(130, -1)
	progBar.AddCSSClass("osd")

	infoBox.Append(titleLbl)
	infoBox.Append(progBar)

	isPlaying := false
	mCopy := msg
	c.trackWidgetSignal(playBtn, playBtn, playBtn.ConnectClicked(func() {
		if isPlaying {
			media.StopCurrentAudio()
			return
		}
		go func() {
			path, err := media.DownloadMedia(context.Background(), c.client, mCopy)
			if err != nil {
				return
			}
			glib.IdleAdd(func() {
				media.PlayAudioPreview(path, func(playing bool) {
					isPlaying = playing
					if playing {
						playBtn.SetIconName("media-playback-pause-symbolic")
						progBar.Pulse()
					} else {
						playBtn.SetIconName("media-playback-start-symbolic")
						progBar.SetFraction(0.0)
					}
				})
			})
		}()
	}))

	actionBox.Append(playBtn)
	actionBox.Append(infoBox)
	return actionBox
}
