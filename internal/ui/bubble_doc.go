package ui

import (
	"strings"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotk4/pkg/pango"
	"postbetasoftware.com/whatsdown/internal/media"
	"postbetasoftware.com/whatsdown/internal/store"
)

func (c *MainController) buildDocWidget(msg store.StoredMessage) *gtk.Box {
	docName := "Document"
	docSize := ""
	isPDF := false
	if e2e := media.ParseE2EMessage(msg.Raw); e2e != nil && e2e.GetDocumentMessage() != nil {
		doc := e2e.GetDocumentMessage()
		if doc.GetFileName() != "" {
			docName = doc.GetFileName()
		}
		docSize = media.FormatFileSize(doc.GetFileLength())
		if strings.HasSuffix(strings.ToLower(docName), ".pdf") || strings.Contains(doc.GetMimetype(), "pdf") {
			isPDF = true
		}
	}

	container := gtk.NewBox(gtk.OrientationVertical, 4)
	mCopy := msg

	if isPDF {
		overlay := gtk.NewOverlay()
		pic := gtk.NewPicture()
		pic.SetKeepAspectRatio(true)
		pic.SetCanShrink(true)
		pic.AddCSSClass("chat-bubble-picture")
		pic.SetSizeRequest(240, 160)
		media.LoadMediaPreviewContext(c.messagePreviewContext, c.client, msg, pic, nil)
		overlay.SetChild(pic)

		badge := gtk.NewLabel("PDF")
		badge.AddCSSClass("media-video-badge")
		badge.SetHAlign(gtk.AlignStart)
		badge.SetVAlign(gtk.AlignStart)
		badge.SetMarginStart(6)
		badge.SetMarginTop(6)
		overlay.AddOverlay(badge)

		gesture := gtk.NewGestureClick()
		c.trackWidgetSignal(overlay, gesture, gesture.ConnectPressed(func(nPress int, x, y float64) {
			c.openPDFPreview(mCopy, docName)
		}))
		overlay.AddController(gesture)
		container.Append(overlay)
	}

	actionBox := gtk.NewBox(gtk.OrientationHorizontal, 6)
	openBtn := gtk.NewButton()
	openBtn.AddCSSClass("flat")

	btnInner := gtk.NewBox(gtk.OrientationHorizontal, 6)
	btnInner.Append(gtk.NewImageFromIconName("folder-documents-symbolic"))
	labelTxt := docName
	if docSize != "" {
		labelTxt += " (" + docSize + ")"
	}
	lbl := gtk.NewLabel(labelTxt)
	lbl.AddCSSClass("caption")
	lbl.SetEllipsize(pango.EllipsizeEnd)
	lbl.SetMaxWidthChars(28)
	btnInner.Append(lbl)
	openBtn.SetChild(btnInner)

	c.trackWidgetSignal(openBtn, openBtn, openBtn.ConnectClicked(func() {
		if isPDF {
			c.openPDFPreview(mCopy, docName)
		} else {
			media.OpenMedia(c.client, mCopy, nil)
		}
	}))
	actionBox.Append(openBtn)
	container.Append(actionBox)

	return container
}
