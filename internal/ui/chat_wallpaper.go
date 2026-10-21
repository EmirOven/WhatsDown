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

	css := buildWallpaperCSS(wp)
	c.chatBgCSSProvider.LoadFromData(css)
}

func buildWallpaperCSS(wp store.ChatWallpaper) string {
	bgCSS, bubbleBg, bubbleFg := extractWallpaperStyle(wp)
	if bgCSS == "" {
		return `.chat-page { background: none; }`
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
