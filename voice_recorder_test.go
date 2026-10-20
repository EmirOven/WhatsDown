package main

import (
	"testing"
	"time"
)

func TestVoiceWaveform(t *testing.T) {
	levels := []float64{0.1, 0.5, 0.8, 1.0, 0.4, 0.2, 0.0}
	waveform := buildVoiceWaveform(levels, 32)
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
	// Silence should give 0
	silence := make([]byte, 1000)
	if lvl := pcmLevel(silence); lvl != 0 {
		t.Fatalf("expected silence level 0, got %f", lvl)
	}
}

func TestVoiceRecorderState(t *testing.T) {
	r := &voiceRecorder{}
	if r.Duration() != 0 {
		t.Fatalf("expected 0 duration")
	}
	r.pcm = make([]byte, voiceSampleRate*voiceBytesPerSample*2) // 2 seconds
	if dur := r.Duration(); dur != 2*time.Second {
		t.Fatalf("expected 2s duration, got %v", dur)
	}
}
