package ui

import (
	"fmt"

	"github.com/diamondburned/gotk4/pkg/glib/v2"
)

// stopVoiceRecording ends audio capture and presents delete/send controls.
func (c *MainController) stopVoiceRecording() {
	if c.voiceRecorder == nil {
		return
	}
	c.voiceRecorder.Stop()

	if c.voiceTimerHandle != 0 {
		glib.SourceRemove(c.voiceTimerHandle)
		c.voiceTimerHandle = 0
	}

	dur := c.voiceRecorder.Duration()
	s := int(dur.Seconds())
	if c.voiceTimerLabel != nil {
		c.voiceTimerLabel.SetText(fmt.Sprintf("%d:%02d", s/60, s%60))
	}
	if c.voicePauseBtn != nil {
		c.voicePauseBtn.SetVisible(false)
	}
	if c.voiceStopBtn != nil {
		c.voiceStopBtn.SetVisible(false)
	}
	if c.voiceStatusIcon != nil {
		c.voiceStatusIcon.SetFromIconName("audio-x-generic-symbolic")
	}
	if c.voicePlayBtn != nil {
		c.voicePlayBtn.SetIconName("media-playback-start-symbolic")
		c.voicePlayBtn.SetVisible(true)
	}
	if c.voiceSendBtn != nil {
		c.voiceSendBtn.SetVisible(true)
	}
	if c.voiceMeterArea != nil {
		c.voiceMeterArea.QueueDraw()
	}
}

// deleteVoiceRecording discards the active or stopped recording.
func (c *MainController) deleteVoiceRecording() {
	c.stopVoicePreview()

	if c.voiceTimerHandle != 0 {
		glib.SourceRemove(c.voiceTimerHandle)
		c.voiceTimerHandle = 0
	}

	if c.voiceRecorder != nil {
		c.voiceRecorder.Stop()
		c.voiceRecorder = nil
	}

	if c.chatInputStack != nil {
		c.chatInputStack.SetVisibleChildName("normal")
	}
	if c.voiceMeterArea != nil {
		c.voiceMeterArea.QueueDraw()
	}
}
