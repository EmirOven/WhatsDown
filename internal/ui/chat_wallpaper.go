package ui

import (
	"context"
	"fmt"
	"strings"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/store"
)

// applyChatWallpaper applies custom background styling and adjusts incoming bubble contrast.
func (c *MainController) applyChatWallpaper(chatJID string) {
	if c.chatPage == nil || c.store == nil {
		return
	}

	if c.chatBgCSSProvider == nil {
		c.chatBgCSSProvider = gtk.NewCSSProvider()
		if display := gdk.DisplayGetDefault(); display != nil {
			gtk.StyleContextAddProviderForDisplay(display, c.chatBgCSSProvider, gtk.STYLE_PROVIDER_PRIORITY_APPLICATION+20)
		}
	}

	wp, err := c.store.GetChatWallpaper(context.Background(), chatJID)
	if err != nil || wp.Kind == "" || wp.Kind == "none" {
		c.chatBgCSSProvider.LoadFromData(`.chat-page { background: none; }`)
		return
	}

	css := buildWallpaperCSS(wp)
	c.chatBgCSSProvider.LoadFromData(css)
}

func buildWallpaperCSS(wp store.ChatWallpaper) string {
	var bgCSS string
	var isLight bool

	switch wp.Kind {
	case "color":
		if wp.Value != "" {
			bgCSS = fmt.Sprintf("background-color: %s; background-image: none;", wp.Value)
			if rgba, ok := parseHexColor(wp.Value); ok {
				isLight = isColorLight(int(rgba.Red()*255), int(rgba.Green()*255), int(rgba.Blue()*255))
			}
		}
	case "image":
		if wp.Value != "" {
			urlVal := wp.Value
			if !strings.HasPrefix(urlVal, "file://") {
				urlVal = "file://" + urlVal
			}
			bgCSS = fmt.Sprintf("background-image: url(\"%s\"); background-size: cover; background-position: center; background-repeat: no-repeat;", urlVal)
			if r, g, b, ok := computeAverageImageColor(wp.Value); ok {
				isLight = isColorLight(r, g, b)
			}
		}
	}

	if bgCSS == "" {
		return `.chat-page { background: none; }`
	}

	var bubbleBg, bubbleFg string
	if isLight {
		bubbleBg = "rgba(0, 0, 0, 0.72)"
		bubbleFg = "#ffffff"
	} else {
		bubbleBg = "rgba(255, 255, 255, 0.88)"
		bubbleFg = "#000000"
	}

	return fmt.Sprintf(`
.chat-page {
  %s
}
.chat-page .chat-bubble-incoming {
  background-color: %s !important;
  color: %s !important;
}
.chat-page .chat-bubble-incoming label,
.chat-page .chat-bubble-incoming .chat-bubble-text,
.chat-page .chat-bubble-incoming .chat-bubble-time,
.chat-page .chat-bubble-incoming .chat-bubble-status,
.chat-page .chat-bubble-incoming .pin-indicator {
  color: %s !important;
}
`, bgCSS, bubbleBg, bubbleFg, bubbleFg)
}
