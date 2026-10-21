package ui

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"postbetasoftware.com/whatsdown/internal/media"
)

// togglePlayVoicePreview allows playing back the stopped recording before sending.
func (c *MainController) togglePlayVoicePreview() {
	if c.voiceRecorder == nil {
		return
	}
	if c.voiceIsPlaying {
		c.stopVoicePreview()
		return
	}
	if c.voicePreviewPath == "" {
		tmpPath := filepath.Join(os.TempDir(), fmt.Sprintf("preview_%d.ogg", time.Now().UnixNano()))
		if err := c.voiceRecorder.EncodeOpus(tmpPath); err != nil {
			log.Printf("encoding voice preview: %v", err)
			return
		}
		c.voicePreviewPath = tmpPath
	}

	c.voiceIsPlaying = true
	if c.voicePlayBtn != nil {
		c.voicePlayBtn.SetIconName("media-playback-pause-symbolic")
	}

	media.PlayAudioPreview(c.voicePreviewPath, func(playing bool) {
		c.voiceIsPlaying = playing
		if !playing {
			if c.voicePlayBtn != nil {
				c.voicePlayBtn.SetIconName("media-playback-start-symbolic")
			}
		}
	})
}

// stopVoicePreview stops playback of the recorded voice note preview.
func (c *MainController) stopVoicePreview() {
	if c.voiceIsPlaying {
		media.StopCurrentAudio()
		c.voiceIsPlaying = false
	}
	if c.voicePlayBtn != nil {
		c.voicePlayBtn.SetIconName("media-playback-start-symbolic")
	}
	if c.voicePreviewPath != "" {
		_ = os.Remove(c.voicePreviewPath)
		c.voicePreviewPath = ""
	}
}
