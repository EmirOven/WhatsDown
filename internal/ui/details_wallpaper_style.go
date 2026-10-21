package ui

import (
	"fmt"
	"strings"

	"postbetasoftware.com/whatsdown/internal/store"
)

func extractWallpaperStyle(wp store.ChatWallpaper) (bgStyle, bubbleBg, bubbleFg string) {
	bubbleBg, bubbleFg = "#ffffff", "#000000"
	switch wp.Kind {
	case "color":
		if wp.Value != "" {
			bgStyle = fmt.Sprintf("background-color: %s; background-image: none;", wp.Value)
			if rgba, ok := parseColorSpec(wp.Value); ok {
				bubbleBg, bubbleFg = wallpaperBubbleColors(int(rgba.Red()*255), int(rgba.Green()*255), int(rgba.Blue()*255))
			}
			return bgStyle, bubbleBg, bubbleFg
		}
	case "image":
		if wp.Value != "" {
			urlVal := wp.Value
			if !strings.HasPrefix(urlVal, "file://") {
				urlVal = "file://" + urlVal
			}
			bgStyle = fmt.Sprintf("background-image: url(\"%s\"); background-size: cover; background-position: center; background-repeat: no-repeat;", urlVal)
			if r, g, b, ok := computeAverageImageColor(wp.Value); ok {
				bubbleBg, bubbleFg = wallpaperBubbleColors(r, g, b)
			}
			return bgStyle, bubbleBg, bubbleFg
		}
	}
	return "", "", ""
}
