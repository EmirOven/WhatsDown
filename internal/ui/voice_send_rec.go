package ui

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/diamondburned/gotk4/pkg/glib/v2"
)

// sendVoiceRecording stops the recorder, encodes to Opus, and dispatches the voice message.
func (c *MainController) sendVoiceRecording() {
	if c.voiceRecorder == nil || c.currentChat == nil {
		c.deleteVoiceRecording()
		return
	}

	c.voiceRecorder.Stop()
	c.stopVoicePreview()

	if c.voiceTimerHandle != 0 {
		glib.SourceRemove(c.voiceTimerHandle)
		c.voiceTimerHandle = 0
	}

	dur := c.voiceRecorder.Duration()
	if dur < 500*time.Millisecond {
		c.deleteVoiceRecording()
		return
	}

	waveform := c.voiceRecorder.Waveform()
	rec := c.voiceRecorder
	c.voiceRecorder = nil

	if c.chatInputStack != nil {
		c.chatInputStack.SetVisibleChildName("normal")
	}

	tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("voice_msg_%d.ogg", time.Now().UnixNano()))
	go func() {
		if err := rec.EncodeOpus(tmpFile); err != nil {
			log.Printf("encoding voice message: %v", err)
			return
		}
		glib.IdleAdd(func() {
			c.sendVoiceMessage(tmpFile, dur, waveform)
		})
	}()
}
