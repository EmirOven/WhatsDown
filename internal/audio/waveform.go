package audio

import "math"

// Waveform returns the 64-sample (0..100) waveform WhatsApp shows for voice notes.
func (r *Recorder) Waveform() []byte {
	return BuildVoiceWaveform(r.Levels(), WaveformBars)
}

// BuildVoiceWaveform downsamples levels into n buckets scaled to 0..100.
func BuildVoiceWaveform(levels []float64, n int) []byte {
	out := make([]byte, n)
	if len(levels) == 0 {
		return out
	}
	buckets := ResampleLevels(levels, n)
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

// ResampleLevels maps levels onto n buckets, taking the max level per bucket.
func ResampleLevels(levels []float64, n int) []float64 {
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
