package audio_test

import (
	"testing"

	"postbetasoftware.com/whatsdown/internal/audio"
)

func TestVoiceWaveform(t *testing.T) {
	levels := []float64{0.1, 0.5, 0.8, 1.0, 0.4, 0.2, 0.0}
	waveform := audio.BuildVoiceWaveform(levels, 32)
	if len(waveform) != 32 {
		t.Fatalf("expected 32 waveform bars, got %d", len(waveform))
	}
	var maxVal byte
	for _, b := range waveform {
		if b > maxVal {
			maxVal = b
		}
	}
	if maxVal != 100 {
		t.Fatalf("expected peak to be normalized to 100, got %d", maxVal)
	}
}

func TestPCMLevel(t *testing.T) {
	silence := make([]byte, 1000)
	if lvl := audio.PCMLevel(silence); lvl != 0 {
		t.Fatalf("expected silence level 0, got %f", lvl)
	}
}

func TestRecorderState(t *testing.T) {
	r := &audio.Recorder{}
	if r.Duration() != 0 {
		t.Fatalf("expected 0 duration")
	}
}
