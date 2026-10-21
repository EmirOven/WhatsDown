package ui

import (
	"context"
	"fmt"

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

	css := buildWallpaperCSS(wp, c.tintingEnabled)
	c.chatBgCSSProvider.LoadFromData(css)
}

func buildWallpaperCSS(wp store.ChatWallpaper, tinting bool) string {
	bgCSS, bubbleBg, bubbleFg := extractWallpaperStyle(wp)
	if bgCSS == "" {
		return `.chat-page { background: none; }`
	}

	var tintCSS string
	if tinting && wp.Kind == "color" && wp.Value != "" {
		tintColor := wp.Value
		var fgColor string
		if rgba, ok := parseColorSpec(tintColor); ok {
			r, g, b := int(rgba.Red()*255), int(rgba.Green()*255), int(rgba.Blue()*255)
			if r > 170 || g > 170 || b > 170 {
				fgColor = "#000000"
			} else {
				fgColor = "#ffffff"
			}
		} else {
			fgColor = "currentColor"
		}
		tintCSS = fmt.Sprintf(`
.right-chat-header,
.right-chat-header windowhandle,
.right-chat-header windowcontrols,
.right-chat-header windowcontrols button {
  background-color: %s;
  color: %s;
}
.right-chat-header label,
.right-chat-header .active-chat-btn label {
  color: %s;
}
.chat-page .chat-input-bar,
.chat-page .voice-recording-bar {
  background-color: %s;
}
.chat-page .chat-input-bar .chat-send-btn {
  background-color: var(--accent-bg-color);
  color: var(--accent-fg-color);
}
`, tintColor, fgColor, fgColor, tintColor)
	}

	return fmt.Sprintf(`
.chat-page {
  %s
}
.chat-page .chat-bubble-incoming {
  background-color: %s;
  color: %s;
}
.chat-page .chat-bubble-incoming label,
.chat-page .chat-bubble-incoming .chat-bubble-text,
.chat-page .chat-bubble-incoming .chat-bubble-time,
.chat-page .chat-bubble-incoming .chat-bubble-status,
.chat-page .chat-bubble-incoming .pin-indicator {
  color: %s;
}
%s
`, bgCSS, bubbleBg, bubbleFg, bubbleFg, tintCSS)
}
