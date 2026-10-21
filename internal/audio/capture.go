package audio

import (
	"errors"
	"fmt"
	"io"
	"os/exec"
)

func captureCommand() (*exec.Cmd, error) {
	if _, err := exec.LookPath("gst-launch-1.0"); err == nil {
		return exec.Command("gst-launch-1.0", "-q",
			"autoaudiosrc", "!", "audioconvert", "!", "audioresample", "!",
			fmt.Sprintf("audio/x-raw,format=S16LE,channels=1,rate=%d", SampleRate), "!",
			"fdsink", "fd=1"), nil
	}
	if _, err := exec.LookPath("ffmpeg"); err == nil {
		return exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error",
			"-f", "pulse", "-i", "default",
			"-ac", "1", "-ar", fmt.Sprint(SampleRate), "-f", "s16le", "pipe:1"), nil
	}
	if _, err := exec.LookPath("pw-record"); err == nil {
		return exec.Command("pw-record", "--format", "s16", "--rate", fmt.Sprint(SampleRate),
			"--channels", "1", "-"), nil
	}
	return nil, errors.New("no audio capture tool found (install GStreamer, ffmpeg or PipeWire tools)")
}

func (r *Recorder) startCapture() error {
	cmd, err := captureCommand()
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting audio capture: %w", err)
	}

	r.mu.Lock()
	r.cmd = cmd
	gen := r.gen
	r.mu.Unlock()

	go r.readLoop(cmd, stdout, gen)
	return nil
}

func (r *Recorder) readLoop(cmd *exec.Cmd, stdout io.Reader, gen int) {
	buf := make([]byte, LevelChunk)
	for {
		n, err := stdout.Read(buf)
		if n > 0 {
			r.mu.Lock()
			if r.gen != gen {
				r.mu.Unlock()
				break
			}
			r.appendPCM(buf[:n])
			r.mu.Unlock()
		}
		if err != nil {
			break
		}
	}
	_ = cmd.Wait()
}

func (r *Recorder) appendPCM(data []byte) {
	r.pcm = append(r.pcm, data...)
	r.pending = append(r.pending, data...)
	for len(r.pending) >= LevelChunk {
		r.levels = append(r.levels, PCMLevel(r.pending[:LevelChunk]))
		r.pending = r.pending[LevelChunk:]
	}
}

func (r *Recorder) stopCapture() {
	r.mu.Lock()
	cmd := r.cmd
	r.cmd = nil
	r.gen++
	r.mu.Unlock()
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}
