package audio

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
)

// EncodeOpus writes captured audio to path as Ogg/Opus.
func (r *Recorder) EncodeOpus(path string) error {
	r.mu.Lock()
	pcm := make([]byte, len(r.pcm))
	copy(pcm, r.pcm)
	r.mu.Unlock()

	if len(pcm) == 0 {
		return errors.New("recording is empty")
	}

	var cmd *exec.Cmd
	if _, err := exec.LookPath("gst-launch-1.0"); err == nil {
		cmd = exec.Command("gst-launch-1.0", "-q",
			"fdsrc", "fd=0", "!",
			"rawaudioparse", "use-sink-caps=false", "format=pcm", "pcm-format=s16le",
			fmt.Sprintf("sample-rate=%d", SampleRate), "num-channels=1", "!",
			"audioconvert", "!", "opusenc", "bitrate=32000", "!", "oggmux", "!",
			"filesink", "location="+path)
	} else if _, err := exec.LookPath("ffmpeg"); err == nil {
		cmd = exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error", "-y",
			"-f", "s16le", "-ar", fmt.Sprint(SampleRate), "-ac", "1", "-i", "pipe:0",
			"-c:a", "libopus", "-b:a", "32k", "-application", "voip", path)
	} else {
		return errors.New("no Opus encoder found (install GStreamer or ffmpeg)")
	}

	var stderr bytes.Buffer
	cmd.Stdin = bytes.NewReader(pcm)
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("encoding voice note: %w: %s", err, stderr.String())
	}
	if fi, err := os.Stat(path); err != nil || fi.Size() == 0 {
		return errors.New("encoding voice note produced no output")
	}
	return nil
}
