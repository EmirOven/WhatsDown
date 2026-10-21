package ui

import (
	"context"
	"fmt"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// applyChatWallpaper applies custom background styling (none, solid color, or image) to the chat view.
func (c *MainController) applyChatWallpaper(chatJID string) {
	if c.chatPage == nil || c.store == nil {
		return
	}

	if c.chatBgCSSProvider == nil {
		c.chatBgCSSProvider = gtk.NewCSSProvider()
		c.chatPage.StyleContext().AddProvider(c.chatBgCSSProvider, gtk.STYLE_PROVIDER_PRIORITY_APPLICATION)
	}

	wp, err := c.store.GetChatWallpaper(context.Background(), chatJID)
	if err != nil || wp.Kind == "" || wp.Kind == "none" {
		c.chatBgCSSProvider.LoadFromData(`.chat-page { background: none; }`)
		return
	}

	switch wp.Kind {
	case "color":
		if wp.Value != "" {
			css := fmt.Sprintf(`.chat-page { background-color: %s; background-image: none; }`, wp.Value)
			c.chatBgCSSProvider.LoadFromData(css)
			return
		}
	case "image":
		if wp.Value != "" {
			css := fmt.Sprintf(`.chat-page { background-image: url("%s"); background-size: cover; background-position: center; background-repeat: no-repeat; }`, wp.Value)
			c.chatBgCSSProvider.LoadFromData(css)
			return
		}
	}

	c.chatBgCSSProvider.LoadFromData(`.chat-page { background: none; }`)
}
