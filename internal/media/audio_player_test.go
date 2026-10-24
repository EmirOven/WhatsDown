package media

import (
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAudioPlayerSeekPauseAndSpeed(t *testing.T) {
	// Real decoding and clocked playback, using a silent sink instead of speakers.
	path := filepath.Join(t.TempDir(), "voice.wav")
	const sampleRate, seconds = 8000, 8
	data := make([]byte, 44+sampleRate*seconds*2)
	copy(data, "RIFF")
	binary.LittleEndian.PutUint32(data[4:], uint32(len(data)-8))
	copy(data[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(data[16:], 16)
	binary.LittleEndian.PutUint16(data[20:], 1)
	binary.LittleEndian.PutUint16(data[22:], 1)
	binary.LittleEndian.PutUint32(data[24:], sampleRate)
	binary.LittleEndian.PutUint32(data[28:], sampleRate*2)
	binary.LittleEndian.PutUint16(data[32:], 2)
	binary.LittleEndian.PutUint16(data[34:], 16)
	copy(data[36:], "data")
	binary.LittleEndian.PutUint32(data[40:], uint32(len(data)-44))
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	p, err := newAudioPlayer(path, "fakesink")
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	deadline := time.Now().Add(3 * time.Second)
	for p.State().Duration == 0 {
		if state := p.State(); state.Err != nil {
			t.Fatal(state.Err)
		}
		if time.Now().After(deadline) {
			t.Fatal("decoder did not prepare WAV")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if d := p.State().Duration; d < 7900*time.Millisecond || d > 8100*time.Millisecond {
		t.Fatalf("duration = %v", d)
	}
	for _, rate := range []float64{0.5, 1, 1.5, 2} {
		p.Pause()
		if err := p.Seek(time.Second, rate); err != nil {
			t.Fatal(err)
		}
		time.Sleep(30 * time.Millisecond)
		if pos := p.State().Position; math.Abs(pos.Seconds()-1) > 0.1 {
			t.Fatalf("seek at %gx: %v", rate, pos)
		}
		if err := p.Play(); err != nil {
			t.Fatal(err)
		}
		start := time.Now()
		time.Sleep(220 * time.Millisecond)
		state := p.State()
		elapsed := time.Since(start).Seconds()
		if !state.Playing || state.Err != nil {
			t.Fatalf("playback: %+v", state)
		}
		advance := state.Position.Seconds() - 1
		if math.Abs(advance-elapsed*rate) > 0.14 {
			t.Fatalf("%gx: advanced %.3fs in %.3fs", rate, advance, elapsed)
		}
	}
	p.Pause()
	time.Sleep(30 * time.Millisecond)
	a := p.State().Position
	time.Sleep(80 * time.Millisecond)
	if state := p.State(); state.Playing || state.Position-a > 20*time.Millisecond {
		t.Fatalf("pause did not retain position: %+v", state)
	}
	p.Close()
	p.Close()
	if err := p.Play(); err == nil {
		t.Fatal("closed player played")
	}
}
