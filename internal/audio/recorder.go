package audio

import (
	"os/exec"
	"sync"
)

const (
	SampleRate     = 48000
	BytesPerSample = 2
	LevelChunk     = SampleRate * BytesPerSample / 20 // 50ms chunk
	WaveformBars   = 64
)

// Recorder captures microphone audio for voice messages.
type Recorder struct {
	mu      sync.Mutex
	pcm     []byte
	levels  []float64
	pending []byte
	cmd     *exec.Cmd
	gen     int
	paused  bool
	stopped bool
}

// NewRecorder creates a recorder and starts capturing.
func NewRecorder() (*Recorder, error) {
	r := &Recorder{}
	if err := r.startCapture(); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *Recorder) Pause() {
	r.mu.Lock()
	if r.paused || r.stopped {
		r.mu.Unlock()
		return
	}
	r.paused = true
	r.mu.Unlock()
	r.stopCapture()
}

func (r *Recorder) Resume() error {
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

func (r *Recorder) Stop() {
	r.mu.Lock()
	r.stopped = true
	r.mu.Unlock()
	r.stopCapture()
}
