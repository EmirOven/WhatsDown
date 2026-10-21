package ui

import (
	"context"
	"log"

	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotk4/pkg/pango"
	"postbetasoftware.com/whatsdown/internal/media"
	"postbetasoftware.com/whatsdown/internal/store"
)

// openPDFPreview downloads the PDF if needed and presents the in-app PDF preview dialog.
func (c *MainController) openPDFPreview(msg store.StoredMessage, docName string) {
	go func() {
		path, err := media.DownloadMedia(context.Background(), c.client, msg)
		if err != nil {
			log.Printf("downloading PDF for preview error: %v", err)
			return
		}
		glib.IdleAdd(func() {
			c.showPDFViewerDialog(path, docName)
		})
	}()
}

// showPDFViewerDialog displays an in-app viewer for a PDF document with a page switcher.
func (c *MainController) showPDFViewerDialog(filePath, docName string) {
	if filePath == "" {
		return
	}
	totalPages := media.GetPDFPageCount(filePath)
	currentPage := 1

	dialog := gtk.NewWindow()
	dialog.SetTransientFor(&c.window.Window)
	dialog.SetModal(true)
	dialog.SetDefaultSize(600, 720)

	headerBar := gtk.NewHeaderBar()
	titleBox := gtk.NewBox(gtk.OrientationVertical, 0)
	titleLbl := gtk.NewLabel(docName)
	titleLbl.AddCSSClass("heading")
	titleLbl.SetEllipsize(pango.EllipsizeEnd)
	titleBox.Append(titleLbl)
	headerBar.SetTitleWidget(titleBox)

	openExtBtn := gtk.NewButton()
	openExtBtn.SetTooltipText("Open in Default App")
	openExtBox := gtk.NewBox(gtk.OrientationHorizontal, 4)
	openExtBox.Append(gtk.NewImageFromIconName("document-open-symbolic"))
	openExtBox.Append(gtk.NewLabel("Open"))
	openExtBtn.SetChild(openExtBox)
	openExtBtn.ConnectClicked(func() {
		_ = media.OpenFile(filePath)
	})
	headerBar.PackEnd(openExtBtn)
	dialog.SetTitlebar(headerBar)

	mainBox := gtk.NewBox(gtk.OrientationVertical, 8)
	mainBox.SetMarginTop(12)
	mainBox.SetMarginBottom(12)
	mainBox.SetMarginStart(16)
	mainBox.SetMarginEnd(16)

	scroll := gtk.NewScrolledWindow()
	scroll.SetPolicy(gtk.PolicyAutomatic, gtk.PolicyAutomatic)
	scroll.SetVExpand(true)
	scroll.SetHExpand(true)

	pic := gtk.NewPicture()
	pic.SetKeepAspectRatio(true)
	pic.SetCanShrink(true)
	pic.SetHExpand(true)
	pic.SetVExpand(true)
	pic.AddCSSClass("pdf-preview-picture")
	scroll.SetChild(pic)
	mainBox.Append(scroll)

	switcherBox := buildPDFSwitcher(filePath, pic, totalPages, &currentPage)
	mainBox.Append(switcherBox)
	dialog.SetChild(mainBox)

	if tex := media.GetPDFPageTexture(filePath, 1); tex != nil {
		pic.SetPaintable(tex)
	} else {
		go func() {
			if tex := media.GetPDFPageTexture(filePath, 1); tex != nil {
				glib.IdleAdd(func() { pic.SetPaintable(tex) })
			}
		}()
	}

	dialog.Present()
}
