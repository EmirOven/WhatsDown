package ui

import (
	"context"
	"fmt"
	"time"

	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/media"
	"postbetasoftware.com/whatsdown/internal/store"
)

func (c *MainController) buildAudioWidget(msg store.StoredMessage) *gtk.Box {
	box := gtk.NewBox(gtk.OrientationHorizontal, 8)
	box.AddCSSClass("chat-bubble-audio-box")
	box.AddCSSClass("message-interactive")
	play := gtk.NewButtonFromIconName("media-playback-start-symbolic")
	addCSSClasses(play, "circular", "suggested-action")
	play.SetTooltipText("Play voice message")
	play.SetVAlign(gtk.AlignCenter)
	play.SetSizeRequest(36, 36)
	info := gtk.NewBox(gtk.OrientationVertical, 2)
	info.SetHExpand(true)
	label := gtk.NewLabel("Voice message")
	if msg.Kind == store.MessageKindAudio {
		label.SetText("Audio")
	}
	label.SetXAlign(0)
	label.AddCSSClass("caption")
	scale := gtk.NewScaleWithRange(gtk.OrientationHorizontal, 0, 1, 0.1)
	scale.SetDrawValue(false)
	scale.SetSizeRequest(150, -1)
	scale.SetHExpand(true)
	scale.SetSensitive(false)
	scale.SetTooltipText("Seek in voice message")
	speed := gtk.NewButtonWithLabel("1x")
	speed.SetTooltipText("Playback speed")
	speed.SetVAlign(gtk.AlignCenter)
	speed.AddCSSClass("flat")
	info.Append(label)
	info.Append(scale)
	box.Append(play)
	box.Append(info)
	box.Append(speed)

	var player *media.AudioPlayer
	var playbackError error
	var timer glib.SourceHandle
	alive, loading, updating := true, false, false
	rates := []float64{0.5, 1, 1.5, 2}
	rateIndex, appliedRate := 1, 1.0
	duration := time.Duration(0)
	if e2e := media.ParseE2EMessage(msg.Raw); e2e != nil {
		duration = time.Duration(e2e.GetAudioMessage().GetSeconds()) * time.Second
	}
	formatTime := func(d time.Duration) string { s := int(d.Seconds()); return fmt.Sprintf("%d:%02d", s/60, s%60) }
	label.SetText("0:00 / " + formatTime(duration))
	ctx, cancel := context.WithCancel(context.Background())
	cleanup := func() {
		if !alive {
			return
		}
		alive = false
		cancel()
		if timer != 0 {
			glib.SourceRemove(timer)
			timer = 0
		}
		if player != nil {
			player.Close()
			player = nil
		}
	}
	c.trackWidgetCleanup(box, cleanup)
	c.trackWidgetSignal(box, box, box.ConnectDestroy(cleanup))
	c.trackWidgetSignal(box, box, box.ConnectUnmap(func() {
		if player != nil {
			player.Pause()
		}
	}))
	report := func(err error) {
		label.SetText("Playback failed")
		label.SetTooltipText(err.Error())
		play.SetIconName("media-playback-start-symbolic")
	}
	update := func() bool {
		if !alive || player == nil {
			return false
		}
		state := player.State()
		if state.Err != nil {
			playbackError = state.Err
		}
		if playbackError != nil {
			report(playbackError)
			return true
		}
		if state.Duration > 0 {
			duration = state.Duration
			scale.SetSensitive(true)
			updating = true
			scale.SetRange(0, duration.Seconds())
			scale.SetValue(state.Position.Seconds())
			updating = false
			if appliedRate != rates[rateIndex] {
				if err := player.Seek(state.Position, rates[rateIndex]); err == nil {
					appliedRate = rates[rateIndex]
				}
			}
		}
		label.SetText(formatTime(state.Position) + " / " + formatTime(duration))
		if state.Playing {
			play.SetIconName("media-playback-pause-symbolic")
			play.SetTooltipText("Pause voice message")
		} else {
			play.SetIconName("media-playback-start-symbolic")
			play.SetTooltipText("Play voice message")
		}
		return true
	}
	c.trackWidgetSignal(scale, scale, scale.ConnectValueChanged(func() {
		if updating || player == nil {
			return
		}
		if err := player.Seek(time.Duration(scale.Value()*float64(time.Second)), rates[rateIndex]); err != nil {
			report(err)
		}
	}))
	c.trackWidgetSignal(speed, speed, speed.ConnectClicked(func() {
		rateIndex = (rateIndex + 1) % len(rates)
		speed.SetLabel(fmt.Sprintf("%gx", rates[rateIndex]))
		if player != nil && duration > 0 {
			if err := player.Seek(player.State().Position, rates[rateIndex]); err != nil {
				report(err)
			} else {
				appliedRate = rates[rateIndex]
			}
		}
	}))
	c.trackWidgetSignal(play, play, play.ConnectClicked(func() {
		if loading {
			return
		}
		playbackError = nil
		if player != nil {
			state := player.State()
			if state.Playing {
				player.Pause()
			} else {
				media.ActivateAudioPlayer(player)
				if state.Duration > 0 && state.Position >= state.Duration-time.Millisecond*100 {
					_ = player.Seek(0, rates[rateIndex])
				}
				if err := player.Play(); err != nil {
					report(err)
				}
			}
			update()
			return
		}
		loading = true
		play.SetSensitive(false)
		label.SetText("Loading audio…")
		go func() {
			path, err := media.DownloadMedia(ctx, c.client, msg)
			glib.IdleAdd(func() {
				if !alive {
					return
				}
				loading = false
				play.SetSensitive(true)
				if err != nil {
					report(err)
					return
				}
				player, err = media.NewAudioPlayer(path)
				if err != nil {
					report(err)
					return
				}
				media.ActivateAudioPlayer(player)
				if err := player.Play(); err != nil {
					report(err)
					return
				}
				timer = glib.TimeoutAdd(150, update)
				update()
			})
		}()
	}))
	return box
}
