package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
)

// parseColorSpec parses either a hex string (e.g. "#1e1e1e", "ff5500") or "rgb(...)/rgba(...)".
func parseColorSpec(spec string) (gdk.RGBA, bool) {
	s := strings.TrimSpace(spec)
	if strings.HasPrefix(s, "rgb") {
		var r, g, b int
		var a float32 = 1.0
		if strings.HasPrefix(s, "rgba") {
			_, err := fmt.Sscanf(s, "rgba(%d,%d,%d,%f)", &r, &g, &b, &a)
			if err != nil {
				_, err = fmt.Sscanf(s, "rgba(%d, %d, %d, %f)", &r, &g, &b, &a)
			}
		} else {
			_, err := fmt.Sscanf(s, "rgb(%d,%d,%d)", &r, &g, &b)
			if err != nil {
				_, err = fmt.Sscanf(s, "rgb(%d, %d, %d)", &r, &g, &b)
			}
		}
		return gdk.NewRGBA(float32(r)/255.0, float32(g)/255.0, float32(b)/255.0, a), true
	}
	return parseHexColor(s)
}

// parseHexColor converts a hex string (e.g. "#1e1e1e", "#ff5500", "#123") into a gdk.RGBA value.
func parseHexColor(hex string) (gdk.RGBA, bool) {
	s := strings.TrimPrefix(strings.TrimSpace(hex), "#")
	var r, g, b uint64
	var a uint64 = 255

	switch len(s) {
	case 3:
		full := fmt.Sprintf("%c%c%c%c%c%c", s[0], s[0], s[1], s[1], s[2], s[2])
		val, err := strconv.ParseUint(full, 16, 32)
		if err != nil {
			return gdk.RGBA{}, false
		}
		r = (val >> 16) & 0xFF
		g = (val >> 8) & 0xFF
		b = val & 0xFF
	case 6:
		val, err := strconv.ParseUint(s, 16, 32)
		if err != nil {
			return gdk.RGBA{}, false
		}
		r = (val >> 16) & 0xFF
		g = (val >> 8) & 0xFF
		b = val & 0xFF
	case 8:
		val, err := strconv.ParseUint(s, 16, 32)
		if err != nil {
			return gdk.RGBA{}, false
		}
		r = (val >> 24) & 0xFF
		g = (val >> 16) & 0xFF
		b = (val >> 8) & 0xFF
		a = val & 0xFF
	default:
		return gdk.RGBA{}, false
	}

	return gdk.NewRGBA(float32(r)/255.0, float32(g)/255.0, float32(b)/255.0, float32(a)/255.0), true
}
