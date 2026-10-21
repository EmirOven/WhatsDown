package ui

import "log"

// togglePauseVoiceRecording toggles pause/resume of the active recording.
func (c *MainController) togglePauseVoiceRecording() {
	if c.voiceRecorder == nil || c.voiceRecorder.Stopped() {
		return
	}
	if c.voiceRecorder.Paused() {
		if err := c.voiceRecorder.Resume(); err != nil {
			log.Printf("resuming voice recording: %v", err)
			return
		}
		if c.voicePauseBtn != nil {
			c.voicePauseBtn.SetIconName("media-playback-pause-symbolic")
			c.voicePauseBtn.SetTooltipText("Pause recording")
		}
		if c.voiceStatusIcon != nil {
			c.voiceStatusIcon.SetFromIconName("media-record-symbolic")
		}
	} else {
		c.voiceRecorder.Pause()
		if c.voicePauseBtn != nil {
			c.voicePauseBtn.SetIconName("media-playback-start-symbolic")
			c.voicePauseBtn.SetTooltipText("Resume recording")
		}
		if c.voiceStatusIcon != nil {
			c.voiceStatusIcon.SetFromIconName("media-playback-pause-symbolic")
		}
	}
	if c.voiceMeterArea != nil {
		c.voiceMeterArea.QueueDraw()
	}
}
