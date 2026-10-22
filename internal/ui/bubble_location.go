package ui

import (
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/media"
	"postbetasoftware.com/whatsdown/internal/store"
)

func (c *MainController) buildLocationWidget(msg store.StoredMessage) *gtk.Box {
	actionBox := gtk.NewBox(gtk.OrientationHorizontal, 6)
	openBtn := gtk.NewButton()
	openBtn.AddCSSClass("flat")

	btnInner := gtk.NewBox(gtk.OrientationHorizontal, 6)
	btnInner.Append(gtk.NewImageFromIconName("find-location-symbolic"))
	lbl := gtk.NewLabel("Open Location in Maps")
	lbl.AddCSSClass("caption")
	btnInner.Append(lbl)
	openBtn.SetChild(btnInner)

	mCopy := msg
	c.trackWidgetSignal(openBtn, openBtn, openBtn.ConnectClicked(func() {
		media.OpenMedia(c.client, mCopy, nil)
	}))
	actionBox.Append(openBtn)
	return actionBox
}
