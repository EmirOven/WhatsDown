package audio

import (
	"encoding/binary"
	"math"
)

// PCMLevel returns a perceptual 0..1 loudness level for a chunk of S16LE samples.
func PCMLevel(chunk []byte) float64 {
	n := len(chunk) / BytesPerSample
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
