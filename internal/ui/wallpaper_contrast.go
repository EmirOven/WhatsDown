package ui

import (
	"fmt"

	"github.com/diamondburned/gotk4/pkg/gdkpixbuf/v2"
)

// wallpaperBubbleColors calculates the incoming bubble background color and text color
// based on the wallpaper's RGB values.
// - If any RGB value > 127: shade is 40/256 darker (clamped at 0).
// - If no RGB value > 127: bubble is made lighter (white).
// - If any background RGB value > 170: text becomes black (#000000); otherwise white (#ffffff).
func wallpaperBubbleColors(r, g, b int) (bubbleBg, bubbleFg string) {
	if r > 127 || g > 127 || b > 127 {
		dr := r - 40
		if dr < 0 {
			dr = 0
		}
		dg := g - 40
		if dg < 0 {
			dg = 0
		}
		db := b - 40
		if db < 0 {
			db = 0
		}
		bubbleBg = fmt.Sprintf("#%02x%02x%02x", dr, dg, db)
	} else {
		bubbleBg = "#ffffff"
	}

	if r > 170 || g > 170 || b > 170 {
		bubbleFg = "#000000"
	} else {
		bubbleFg = "#ffffff"
	}

	return bubbleBg, bubbleFg
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

