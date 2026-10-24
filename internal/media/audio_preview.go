package media

import (
	"os/exec"
	"sync"

	"github.com/diamondburned/gotk4/pkg/glib/v2"
)

var (
	currentNativeAudio *AudioPlayer
	currentAudioCmd    *exec.Cmd
	currentAudioPath   string
	currentAudioStop   func()
	audioPlayingMu     sync.Mutex
)

// StopCurrentAudio stops any ongoing audio preview playback.
func StopCurrentAudio() {
	audioPlayingMu.Lock()
	defer audioPlayingMu.Unlock()
	if currentNativeAudio != nil {
		currentNativeAudio.Pause()
		currentNativeAudio = nil
	}
	if currentAudioCmd != nil && currentAudioCmd.Process != nil {
		_ = currentAudioCmd.Process.Kill()
		currentAudioCmd = nil
	}
	if currentAudioStop != nil {
		currentAudioStop()
		currentAudioStop = nil
	}
	currentAudioPath = ""
}

// PlayAudioPreview plays audio for preview using gst-play-1.0 or ffplay.
func PlayAudioPreview(path string, onPlaybackState func(playing bool)) {
	audioPlayingMu.Lock()
	if currentAudioPath == path && currentAudioCmd != nil {
		audioPlayingMu.Unlock()
		StopCurrentAudio()
		return
	}
	audioPlayingMu.Unlock()

	StopCurrentAudio()

	audioPlayingMu.Lock()
	currentAudioPath = path
	currentAudioStop = func() {
		if onPlaybackState != nil {
			glib.IdleAdd(func() {
				onPlaybackState(false)
			})
		}
	}
	audioPlayingMu.Unlock()

	if onPlaybackState != nil {
		onPlaybackState(true)
	}

	go func() {
		var cmd *exec.Cmd
		if _, err := exec.LookPath("gst-play-1.0"); err == nil {
			cmd = exec.Command("gst-play-1.0", "--no-interactive", path)
		} else if _, err := exec.LookPath("ffplay"); err == nil {
			cmd = exec.Command("ffplay", "-nodisp", "-autoexit", path)
		} else {
			cmd = exec.Command("xdg-open", path)
		}

		audioPlayingMu.Lock()
		currentAudioCmd = cmd
		audioPlayingMu.Unlock()

		_ = cmd.Run()

		audioPlayingMu.Lock()
		if currentAudioCmd == cmd {
			currentAudioCmd = nil
			if currentAudioStop != nil {
				currentAudioStop()
				currentAudioStop = nil
			}
			currentAudioPath = ""
		}
		audioPlayingMu.Unlock()
	}()
}
