package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"sync"
	"time"
)

// Voice notes are captured as raw 16-bit mono PCM and kept in memory, which
// makes pausing trivial (the capture process is simply stopped and restarted)
// and lets us compute the live level meter and the WhatsApp waveform in Go.
// The PCM is encoded to Ogg/Opus (the format WhatsApp uses for voice notes)
// once the recording is stopped.
const (
	voiceSampleRate     = 48000
	voiceBytesPerSample = 2
	// voiceLevelChunk is the number of bytes per level-meter sample (50 ms).
	voiceLevelChunk = voiceSampleRate * voiceBytesPerSample / 20
	// voiceWaveformBars is the number of waveform samples WhatsApp expects.
	voiceWaveformBars = 64
)

// voiceRecorder captures microphone audio for a voice note.
type voiceRecorder struct {
	mu      sync.Mutex
	pcm     []byte
	levels  []float64 // one normalized (0..1) level per voiceLevelChunk
	pending []byte    // bytes not yet accounted for in levels
	cmd     *exec.Cmd
	gen     int // incremented every time capture stops; stale readers drop data
	paused  bool
	stopped bool
}

// newVoiceRecorder creates a recorder and immediately starts capturing.
func newVoiceRecorder() (*voiceRecorder, error) {
	r := &voiceRecorder{}
	if err := r.startCapture(); err != nil {
		return nil, err
	}
	return r, nil
}

// captureCommand returns a command that writes raw S16LE mono 48 kHz PCM from
// the default microphone to stdout.
func captureCommand() (*exec.Cmd, error) {
	if _, err := exec.LookPath("gst-launch-1.0"); err == nil {
		return exec.Command("gst-launch-1.0", "-q",
			"autoaudiosrc", "!", "audioconvert", "!", "audioresample", "!",
			fmt.Sprintf("audio/x-raw,format=S16LE,channels=1,rate=%d", voiceSampleRate), "!",
			"fdsink", "fd=1"), nil
	}
	if _, err := exec.LookPath("ffmpeg"); err == nil {
		return exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error",
			"-f", "pulse", "-i", "default",
			"-ac", "1", "-ar", fmt.Sprint(voiceSampleRate), "-f", "s16le", "pipe:1"), nil
	}
	if _, err := exec.LookPath("pw-record"); err == nil {
		return exec.Command("pw-record", "--format", "s16", "--rate", fmt.Sprint(voiceSampleRate),
			"--channels", "1", "-"), nil
	}
	return nil, errors.New("no audio capture tool found (install GStreamer, ffmpeg or PipeWire tools)")
}

func (r *voiceRecorder) startCapture() error {
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

func (r *voiceRecorder) readLoop(cmd *exec.Cmd, stdout io.Reader, gen int) {
	buf := make([]byte, voiceLevelChunk)
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

// appendPCM stores captured samples and updates the level meter. Caller holds r.mu.
func (r *voiceRecorder) appendPCM(data []byte) {
	r.pcm = append(r.pcm, data...)
	r.pending = append(r.pending, data...)
	for len(r.pending) >= voiceLevelChunk {
		r.levels = append(r.levels, pcmLevel(r.pending[:voiceLevelChunk]))
		r.pending = r.pending[voiceLevelChunk:]
	}
}

// pcmLevel returns a perceptual 0..1 loudness level for a chunk of S16LE samples.
func pcmLevel(chunk []byte) float64 {
	n := len(chunk) / voiceBytesPerSample
	if n == 0 {
		return 0
	}
	var sum float64
	for i := 0; i < n; i++ {
		s := float64(int16(binary.LittleEndian.Uint16(chunk[i*2:])))
		sum += s * s
	}
	rms := math.Sqrt(sum/float64(n)) / 32768
	if rms <= 0 {
		return 0
	}
	// Map -50 dBFS..0 dBFS to 0..1.
	level := (20*math.Log10(rms) + 50) / 50
	return math.Max(0, math.Min(1, level))
}

// stopCapture terminates the running capture process, if any.
func (r *voiceRecorder) stopCapture() {
	r.mu.Lock()
	cmd := r.cmd
	r.cmd = nil
	r.gen++
	r.mu.Unlock()
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}

// Pause stops capturing (releasing the microphone) without discarding audio.
func (r *voiceRecorder) Pause() {
	r.mu.Lock()
	if r.paused || r.stopped {
		r.mu.Unlock()
		return
	}
	r.paused = true
	r.mu.Unlock()
	r.stopCapture()
}

// Resume continues a paused recording.
func (r *voiceRecorder) Resume() error {
	r.mu.Lock()
	if !r.paused || r.stopped {
		r.mu.Unlock()
		return nil
	}
	r.paused = false
	r.mu.Unlock()
	if err := r.startCapture(); err != nil {
		r.mu.Lock()
		r.paused = true
		r.mu.Unlock()
		return err
	}
	return nil
}

// Stop ends the recording for good.
func (r *voiceRecorder) Stop() {
	r.mu.Lock()
	r.stopped = true
	r.mu.Unlock()
	r.stopCapture()
}

// Paused reports whether the recording is currently paused.
func (r *voiceRecorder) Paused() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.paused
}

// Duration returns the length of the captured audio.
func (r *voiceRecorder) Duration() time.Duration {
	r.mu.Lock()
	defer r.mu.Unlock()
	samples := len(r.pcm) / voiceBytesPerSample
	return time.Duration(samples) * time.Second / voiceSampleRate
}

// Levels returns a snapshot of the per-50ms loudness levels.
func (r *voiceRecorder) Levels() []float64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]float64, len(r.levels))
	copy(out, r.levels)
	return out
}

// Waveform returns the 64-sample (0..100) waveform WhatsApp shows for voice notes.
func (r *voiceRecorder) Waveform() []byte {
	return buildVoiceWaveform(r.Levels(), voiceWaveformBars)
}

// buildVoiceWaveform downsamples levels into n buckets scaled to 0..100.
func buildVoiceWaveform(levels []float64, n int) []byte {
	out := make([]byte, n)
	if len(levels) == 0 {
		return out
	}
	buckets := resampleLevels(levels, n)
	peak := 0.0
	for _, v := range buckets {
		peak = math.Max(peak, v)
	}
	if peak == 0 {
		return out
	}
	for i, v := range buckets {
		out[i] = byte(math.Round(v / peak * 100))
	}
	return out
}

// resampleLevels maps levels onto n buckets, taking the max level per bucket.
func resampleLevels(levels []float64, n int) []float64 {
	out := make([]float64, n)
	if len(levels) == 0 || n == 0 {
		return out
	}
	for i := 0; i < n; i++ {
		start := i * len(levels) / n
		end := (i + 1) * len(levels) / n
		if end <= start {
			end = start + 1
		}
		if end > len(levels) {
			end = len(levels)
		}
		for _, v := range levels[start:end] {
			out[i] = math.Max(out[i], v)
		}
	}
	return out
}

// EncodeOpus writes the captured audio to path as Ogg/Opus.
func (r *voiceRecorder) EncodeOpus(path string) error {
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
			fmt.Sprintf("sample-rate=%d", voiceSampleRate), "num-channels=1", "!",
			"audioconvert", "!", "opusenc", "bitrate=32000", "!", "oggmux", "!",
			"filesink", "location="+path)
	} else if _, err := exec.LookPath("ffmpeg"); err == nil {
		cmd = exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error", "-y",
			"-f", "s16le", "-ar", fmt.Sprint(voiceSampleRate), "-ac", "1", "-i", "pipe:0",
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
