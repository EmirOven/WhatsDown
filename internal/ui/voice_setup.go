package ui

import (
	"github.com/diamondburned/gotk4/pkg/cairo"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// setupVoiceRecording wires up the voice message recording and meter controls.
func (c *MainController) setupVoiceRecording() {
	if c.voiceRecordBtn != nil {
		c.voiceRecordBtn.ConnectClicked(func() {
			c.startVoiceRecording()
		})
	}
	if c.voiceDeleteBtn != nil {
		c.voiceDeleteBtn.ConnectClicked(func() {
			c.deleteVoiceRecording()
		})
	}
	if c.voicePauseBtn != nil {
		c.voicePauseBtn.ConnectClicked(func() {
			c.togglePauseVoiceRecording()
		})
	}
	if c.voiceStopBtn != nil {
		c.voiceStopBtn.ConnectClicked(func() {
			c.stopVoiceRecording()
		})
	}
	if c.voicePlayBtn != nil {
		c.voicePlayBtn.ConnectClicked(func() {
			c.togglePlayVoicePreview()
		})
	}
	if c.voiceSendBtn != nil {
		c.voiceSendBtn.ConnectClicked(func() {
			c.sendVoiceRecording()
		})
	}
	if c.voiceMeterArea != nil {
		c.voiceMeterArea.SetDrawFunc(func(area *gtk.DrawingArea, cr *cairo.Context, width, height int) {
			c.drawVoiceMeter(area, cr, width, height)
		})
	}
}
