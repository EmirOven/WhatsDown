package ui

import (
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/media"
	"postbetasoftware.com/whatsdown/internal/store"
)

func (c *MainController) appendEditContextActions(menuBox *gtk.Box, popover *gtk.Popover, mCopy store.StoredMessage) {
	editBtn := gtk.NewButton()
	addCSSClasses(editBtn, "flat", "popover-menu-btn")
	eBox := gtk.NewBox(gtk.OrientationHorizontal, 10)
	eBox.Append(gtk.NewImageFromIconName("document-edit-symbolic"))
	eBox.Append(gtk.NewLabel("Edit Message"))
	editBtn.SetChild(eBox)
	c.trackWidgetSignal(editBtn, editBtn, editBtn.ConnectClicked(func() {
		popover.Popdown()
		c.startEditing(mCopy)
	}))
	menuBox.Append(editBtn)

	if mCopy.Edited {
		c.appendEditHistoryAction(menuBox, popover, mCopy)
	}
}

func (c *MainController) appendEditHistoryAction(menuBox *gtk.Box, popover *gtk.Popover, mCopy store.StoredMessage) {
	historyBtn := gtk.NewButton()
	addCSSClasses(historyBtn, "flat", "popover-menu-btn")
	hBox := gtk.NewBox(gtk.OrientationHorizontal, 10)
	hBox.Append(gtk.NewImageFromIconName("document-revert-symbolic"))
	hBox.Append(gtk.NewLabel("View Edit History"))
	historyBtn.SetChild(hBox)
	c.trackWidgetSignal(historyBtn, historyBtn, historyBtn.ConnectClicked(func() {
		popover.Popdown()
		c.showEditHistoryDialog(mCopy)
	}))
	menuBox.Append(historyBtn)
}

func (c *MainController) appendStandardContextActions(menuBox *gtk.Box, popover *gtk.Popover, mCopy store.StoredMessage) {
	starText := "Star Message"
	starIcon := "starred-symbolic"
	if mCopy.Starred {
		starText = "Unstar Message"
		starIcon = "non-starred-symbolic"
	}
	starBtn := gtk.NewButton()
	addCSSClasses(starBtn, "flat", "popover-menu-btn")
	sBox := gtk.NewBox(gtk.OrientationHorizontal, 10)
	sBox.Append(gtk.NewImageFromIconName(starIcon))
	sBox.Append(gtk.NewLabel(starText))
	starBtn.SetChild(sBox)
	c.trackWidgetSignal(starBtn, starBtn, starBtn.ConnectClicked(func() {
		popover.Popdown()
		c.toggleMessageStarred(mCopy)
	}))
	menuBox.Append(starBtn)

	copyBtn := gtk.NewButton()
	addCSSClasses(copyBtn, "flat", "popover-menu-btn")
	cBox := gtk.NewBox(gtk.OrientationHorizontal, 10)
	cBox.Append(gtk.NewImageFromIconName("edit-copy-symbolic"))
	cLabel := "Copy Text"
	if mCopy.Kind == store.MessageKindImage || mCopy.Kind == store.MessageKindVideo ||
		mCopy.Kind == store.MessageKindDocument || mCopy.Kind == store.MessageKindAudio ||
		mCopy.Kind == store.MessageKindVoice {
		cLabel = "Copy Media"
	}
	cBox.Append(gtk.NewLabel(cLabel))
	copyBtn.SetChild(cBox)
	c.trackWidgetSignal(copyBtn, copyBtn, copyBtn.ConnectClicked(func() {
		popover.Popdown()
		if mCopy.Kind == store.MessageKindText {
			clipboard := gdk.DisplayGetDefault().Clipboard()
			clipboard.SetText(mCopy.Text)
		} else {
			media.CopyMediaToClipboard(c.client, mCopy)
		}
	}))
	menuBox.Append(copyBtn)

	if mCopy.Kind == store.MessageKindImage || mCopy.Kind == store.MessageKindVideo ||
		mCopy.Kind == store.MessageKindDocument || mCopy.Kind == store.MessageKindAudio ||
		mCopy.Kind == store.MessageKindVoice {
		fmBtn := gtk.NewButton()
		addCSSClasses(fmBtn, "flat", "popover-menu-btn")
		fBox := gtk.NewBox(gtk.OrientationHorizontal, 10)
		fBox.Append(gtk.NewImageFromIconName("folder-open-symbolic"))
		fBox.Append(gtk.NewLabel("Show in File Manager"))
		fmBtn.SetChild(fBox)
		c.trackWidgetSignal(fmBtn, fmBtn, fmBtn.ConnectClicked(func() {
			popover.Popdown()
			media.ShowMediaInFileManager(c.client, mCopy)
		}))
		menuBox.Append(fmBtn)
	}
}
