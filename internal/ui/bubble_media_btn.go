package ui

import (
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/media"
	"postbetasoftware.com/whatsdown/internal/store"
)

func (c *MainController) attachMediaOverlayButton(overlay *gtk.Overlay, msg store.StoredMessage) {
	openBtn := gtk.NewButton()
	addCSSClasses(openBtn, "osd", "pill", "media-overlay-btn")
	openBtn.SetHAlign(gtk.AlignCenter)
	openBtn.SetVAlign(gtk.AlignCenter)

	iconName := "insert-image-symbolic"
	btnText := "View Photo"
	if msg.Kind == store.MessageKindVideo {
		iconName = "media-playback-start-symbolic"
		btnText = "Play Video"
	}
	btnInner := gtk.NewBox(gtk.OrientationHorizontal, 6)
	btnInner.Append(gtk.NewImageFromIconName(iconName))
	lbl := gtk.NewLabel(btnText)
	lbl.AddCSSClass("caption")
	btnInner.Append(lbl)
	openBtn.SetChild(btnInner)

	mCopy := msg
	c.trackWidgetSignal(openBtn, openBtn, openBtn.ConnectClicked(func() {
		media.OpenMedia(c.client, mCopy, func(s string) {
			glib.IdleAdd(func() { lbl.SetText(s) })
		})
	}))
	overlay.AddOverlay(openBtn)

	gesture := gtk.NewGestureClick()
	c.trackWidgetSignal(overlay, gesture, gesture.ConnectPressed(func(nPress int, x, y float64) {
		if nPress != 1 {
			return
		}
		if picked := overlay.Pick(x, y, gtk.PickDefault); picked != nil {
			pw := gtk.BaseWidget(picked)
			if pw.Native() == gtk.BaseWidget(openBtn).Native() || pw.IsAncestor(openBtn) {
				return
			}
		}
		media.OpenMedia(c.client, mCopy, nil)
	}))
	overlay.AddController(gesture)
}
