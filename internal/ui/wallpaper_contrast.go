package ui

import (
	"github.com/diamondburned/gotk4/pkg/gdkpixbuf/v2"
)

// isColorLight checks if any channel R, G, or B is > 127 (lighter than (127,127,127)).
func isColorLight(r, g, b int) bool {
	return r > 127 || g > 127 || b > 127
}

// computeAverageImageColor loads the image via gdkpixbuf and calculates average RGB.
func computeAverageImageColor(path string) (r, g, b int, ok bool) {
	pb, err := gdkpixbuf.NewPixbufFromFileAtScale(path, 64, 64, true)
	if err != nil || pb == nil {
		return 0, 0, 0, false
	}

	pixels := pb.Pixels()
	channels := pb.NChannels()
	if len(pixels) == 0 || channels < 3 {
		return 0, 0, 0, false
	}

	var sumR, sumG, sumB, count uint64
	for i := 0; i+2 < len(pixels); i += channels {
		sumR += uint64(pixels[i])
		sumG += uint64(pixels[i+1])
		sumB += uint64(pixels[i+2])
		count++
	}

	if count == 0 {
		return 0, 0, 0, false
	}

	return int(sumR / count), int(sumG / count), int(sumB / count), true
}
