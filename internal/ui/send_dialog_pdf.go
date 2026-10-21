package ui

import (
	"fmt"

	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/media"
)

func buildPDFSendPreview(filePath string) *gtk.Box {
	box := gtk.NewBox(gtk.OrientationVertical, 8)
	box.SetHAlign(gtk.AlignCenter)

	overlay := gtk.NewOverlay()
	pic := gtk.NewPicture()
	pic.SetKeepAspectRatio(true)
	pic.SetCanShrink(true)
	pic.SetSizeRequest(320, 240)
	pic.AddCSSClass("chat-bubble-picture")
	totalPages := media.GetPDFPageCount(filePath)
	currentPage := 1

	if tex := media.GetPDFPageTexture(filePath, 1); tex != nil {
		pic.SetPaintable(tex)
	} else {
		go func() {
			if tex := media.GetPDFPageTexture(filePath, 1); tex != nil {
				glib.IdleAdd(func() { pic.SetPaintable(tex) })
			}
		}()
	}
	overlay.SetChild(pic)

	badge := gtk.NewLabel("PDF Preview")
	badge.AddCSSClass("media-video-badge")
	badge.SetHAlign(gtk.AlignStart)
	badge.SetVAlign(gtk.AlignStart)
	badge.SetMarginStart(6)
	badge.SetMarginTop(6)
	overlay.AddOverlay(badge)
	box.Append(overlay)

	switcherBox := gtk.NewBox(gtk.OrientationHorizontal, 8)
	switcherBox.SetHAlign(gtk.AlignCenter)
	switcherBox.SetMarginTop(8)
	switcherBox.AddCSSClass("pdf-page-switcher")

	prevBtn := gtk.NewButtonFromIconName("go-previous-symbolic")
	addCSSClasses(prevBtn, "flat", "circular")
	prevBtn.SetSensitive(false)
	prevBtn.SetTooltipText("Previous page")

	pageLbl := gtk.NewLabel(fmt.Sprintf("Page %d of %d", currentPage, totalPages))
	addCSSClasses(pageLbl, "caption", "dim-label")

	nextBtn := gtk.NewButtonFromIconName("go-next-symbolic")
	addCSSClasses(nextBtn, "flat", "circular")
	nextBtn.SetSensitive(totalPages > 1)
	nextBtn.SetTooltipText("Next page")

	updatePage := func(newPage int) {
		if newPage < 1 || newPage > totalPages {
			return
		}
		currentPage = newPage
		pageLbl.SetText(fmt.Sprintf("Page %d of %d", currentPage, totalPages))
		prevBtn.SetSensitive(currentPage > 1)
		nextBtn.SetSensitive(currentPage < totalPages)
		go func() {
			if tex := media.GetPDFPageTexture(filePath, currentPage); tex != nil {
				glib.IdleAdd(func() { pic.SetPaintable(tex) })
			}
		}()
	}

	prevBtn.ConnectClicked(func() { updatePage(currentPage - 1) })
	nextBtn.ConnectClicked(func() { updatePage(currentPage + 1) })

	switcherBox.Append(prevBtn)
	switcherBox.Append(pageLbl)
	switcherBox.Append(nextBtn)
	box.Append(switcherBox)

	return box
}
