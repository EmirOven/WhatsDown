package ui

import (
	"fmt"

	"github.com/diamondburned/gotk4/pkg/gdkpixbuf/v2"
)

// wallpaperBubbleColors calculates the incoming bubble background color and text color
// based on the wallpaper's RGB values:
// - If any RGB value > 127: bubble is shaded 40 darker than the wallpaper background (clamped at 0).
// - If no RGB value > 127 (dark background): bubble is shaded 40 lighter than the wallpaper (clamped at 255).
// - Text color: if bubble background is light (any channel > 170), text is black (#000000); otherwise white (#ffffff).
func wallpaperBubbleColors(r, g, b int) (bubbleBg, bubbleFg string) {
	var br, bg, bb int
	if r > 127 || g > 127 || b > 127 {
		br = r - 40
		if br < 0 {
			br = 0
		}
		bg = g - 40
		if bg < 0 {
			bg = 0
		}
		bb = b - 40
		if bb < 0 {
			bb = 0
		}
	} else {
		br = r + 40
		if br > 255 {
			br = 255
		}
		bg = g + 40
		if bg > 255 {
			bg = 255
		}
		bb = b + 40
		if bb > 255 {
			bb = 255
		}
	}
	bubbleBg = fmt.Sprintf("#%02x%02x%02x", br, bg, bb)

	if br > 170 || bg > 170 || bb > 170 {
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
