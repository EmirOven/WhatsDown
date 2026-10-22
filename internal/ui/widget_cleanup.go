package ui

import (
	coreglib "github.com/diamondburned/gotk4/pkg/core/glib"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// Track callbacks on disposable message widgets. Native parents and Go signal
// closures can otherwise keep each other alive after a row leaves the list.
func (c *MainController) trackWidgetSignal(owner gtk.Widgetter, source coreglib.Objector, handle coreglib.SignalHandle) {
	if c.widgetSignalCleanups == nil {
		c.widgetSignalCleanups = make(map[uintptr][]func())
	}
	key := uintptr(coreglib.BaseObject(owner).Native())
	c.widgetSignalCleanups[key] = append(c.widgetSignalCleanups[key], func() {
		coreglib.BaseObject(source).HandlerDisconnect(handle)
	})
}

// releaseMessageWidget disconnects callbacks without destroying wrappers:
// outstanding media downloads may still finish and update an unparented picture.
func (c *MainController) releaseMessageWidget(widget gtk.Widgetter) {
	w := gtk.BaseWidget(widget)
	key := uintptr(coreglib.BaseObject(widget).Native())
	for _, cleanup := range c.widgetSignalCleanups[key] {
		cleanup()
	}
	delete(c.widgetSignalCleanups, key)
	for child := w.FirstChild(); child != nil; {
		next := gtk.BaseWidget(child).NextSibling()
		c.releaseMessageWidget(child)
		switch child.(type) {
		case *gtk.Popover, *gtk.EmojiChooser:
			gtk.BaseWidget(child).Unparent()
		}
		child = next
	}
}
