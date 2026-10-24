package ui

import (
	"strings"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/media"
	"postbetasoftware.com/whatsdown/internal/store"
)

func (c *MainController) buildDetailsDocsTab(dialog *adw.Dialog, docMsgs []store.StoredMessage) *gtk.Box {
	docsBox := gtk.NewBox(gtk.OrientationVertical, 8)
	docsBox.SetMarginTop(8)
	if len(docMsgs) == 0 {
		emptyGroup := adw.NewPreferencesGroup()
		emptyRow := adw.NewActionRow()
		emptyRow.SetTitle("No documents")
		emptyRow.SetSubtitle("Shared documents and files will appear here")
		emptyRow.AddPrefix(gtk.NewImageFromIconName("folder-documents-symbolic"))
		emptyGroup.Add(emptyRow)
		docsBox.Append(emptyGroup)
		return docsBox
	}

	docGroup := adw.NewPreferencesGroup()
	for _, m := range docMsgs {
		row := adw.NewActionRow()
		titleText := "Document"
		e2e := media.ParseE2EMessage(m.Raw)
		if e2e != nil && e2e.GetDocumentMessage() != nil {
			doc := e2e.GetDocumentMessage()
			if doc.GetFileName() != "" {
				titleText = doc.GetFileName()
			}
			if sz := media.FormatFileSize(doc.GetFileLength()); sz != "" {
				titleText += " • " + sz
			}
		}
		row.AddPrefix(gtk.NewImageFromIconName("folder-documents-symbolic"))
		row.SetTitle(titleText)
		sub := m.Timestamp.Format("02 Jan 15:04")
		if m.FromMe {
			sub = "You • " + sub
		} else if m.SenderName != "" && !strings.Contains(m.SenderName, "@") {
			sub = m.SenderName + " • " + sub
		}
		row.SetSubtitle(sub)

		openBtn := gtk.NewButtonFromIconName("document-open-symbolic")
		openBtn.AddCSSClass("flat")
		openBtn.SetTooltipText("Open")
		openBtn.SetVAlign(gtk.AlignCenter)
		mCopy := m
		openBtn.ConnectClicked(func() {
			if e2e != nil && (strings.Contains(e2e.GetDocumentMessage().GetMimetype(), "pdf") || strings.HasSuffix(strings.ToLower(e2e.GetDocumentMessage().GetFileName()), ".pdf")) {
				dialog.Close()
				c.openMediaViewer(mCopy)
			} else {
				media.OpenMedia(c.client, mCopy, nil)
			}
		})
		row.AddSuffix(openBtn)
		row.SetActivatable(true)
		row.ConnectActivated(func() {
			if e2e != nil && (strings.Contains(e2e.GetDocumentMessage().GetMimetype(), "pdf") || strings.HasSuffix(strings.ToLower(e2e.GetDocumentMessage().GetFileName()), ".pdf")) {
				dialog.Close()
				c.openMediaViewer(mCopy)
			} else {
				media.OpenMedia(c.client, mCopy, nil)
			}
		})

		c.attachMediaContextMenu(row, mCopy, dialog)
		docGroup.Add(row)
	}
	docsBox.Append(docGroup)
	return docsBox
}
