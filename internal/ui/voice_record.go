package ui

import (
	"fmt"
	"log"

	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"postbetasoftware.com/whatsdown/internal/audio"
	"postbetasoftware.com/whatsdown/internal/media"
)

// startVoiceRecording begins audio capture and presents the recording bar.
func (c *MainController) startVoiceRecording() {
	if c.currentChat == nil {
		return
	}
	media.StopCurrentAudio()
	c.stopVoicePreview()

	rec, err := audio.NewRecorder()
	if err != nil {
		log.Printf("startVoiceRecording error: %v", err)
		return
	}
	c.voiceRecorder = rec

	if c.chatInputStack != nil {
		c.chatInputStack.SetVisibleChildName("recording")
	}
	if c.voiceTimerLabel != nil {
		c.voiceTimerLabel.SetText("0:00")
	}
	if c.voiceStatusIcon != nil {
		c.voiceStatusIcon.SetFromIconName("media-record-symbolic")
		c.voiceStatusIcon.SetVisible(true)
	}
	if c.voicePauseBtn != nil {
		c.voicePauseBtn.SetIconName("media-playback-pause-symbolic")
		c.voicePauseBtn.SetTooltipText("Pause recording")
		c.voicePauseBtn.SetVisible(true)
	}
	if c.voiceStopBtn != nil {
		c.voiceStopBtn.SetVisible(true)
	}
	if c.voicePlayBtn != nil {
		c.voicePlayBtn.SetVisible(false)
	}
	if c.voiceSendBtn != nil {
		c.voiceSendBtn.SetVisible(false)
	}

	if c.voiceTimerHandle != 0 {
		glib.SourceRemove(c.voiceTimerHandle)
		c.voiceTimerHandle = 0
	}

	c.voiceTimerHandle = glib.TimeoutAdd(50, func() bool {
		if c.voiceRecorder == nil {
			c.voiceTimerHandle = 0
			return false
		}
		dur := c.voiceRecorder.Duration()
		s := int(dur.Seconds())
		if c.voiceTimerLabel != nil {
			c.voiceTimerLabel.SetText(fmt.Sprintf("%d:%02d", s/60, s%60))
		}
		if c.voiceMeterArea != nil {
			c.voiceMeterArea.QueueDraw()
		}
		if c.voiceRecorder.Stopped() {
			c.voiceTimerHandle = 0
			return false
		}
		return true
	})
}
