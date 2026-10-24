package ui

import (
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/store"
)

func (c *MainController) attachMediaOverlayButton(overlay *gtk.Overlay, msg store.StoredMessage) {
	overlay.SetFocusable(true)
	overlay.SetCursorFromName("pointer")
	overlay.SetTooltipText("Open media")
	click := gtk.NewGestureClick()
	click.SetButton(1)
	c.trackWidgetSignal(overlay, click, click.ConnectReleased(func(n int, _, _ float64) {
		if n == 1 {
			c.openMediaViewer(msg)
		}
	}))
	overlay.AddController(click)
	keys := gtk.NewEventControllerKey()
	c.trackWidgetSignal(overlay, keys, keys.ConnectKeyPressed(func(key, _ uint, _ gdk.ModifierType) bool {
		if key == gdk.KEY_Return || key == gdk.KEY_space {
			c.openMediaViewer(msg)
			return true
		}
		return false
	}))
	overlay.AddController(keys)
}
