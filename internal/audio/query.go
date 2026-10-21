package audio

import "time"

func (r *Recorder) Paused() bool {
	if r == nil {
		return false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.paused
}

func (r *Recorder) Stopped() bool {
	if r == nil {
		return true
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.stopped
}

func (r *Recorder) Duration() time.Duration {
	if r == nil {
		return 0
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	samples := len(r.pcm) / BytesPerSample
	return time.Duration(samples) * time.Second / SampleRate
}

func (r *Recorder) Levels() []float64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]float64, len(r.levels))
	copy(out, r.levels)
	return out
}
