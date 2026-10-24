package ui

import (
	"fmt"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/media"
)

func buildPDFSwitcher(filePath string, setTexture func(*gdk.Texture), totalPages int, currentPage *int) *gtk.Box {
	switcherBox := gtk.NewBox(gtk.OrientationHorizontal, 12)
	switcherBox.SetHAlign(gtk.AlignCenter)
	switcherBox.SetMarginTop(8)
	switcherBox.AddCSSClass("pdf-page-switcher")

	prevBtn := gtk.NewButtonFromIconName("go-previous-symbolic")
	addCSSClasses(prevBtn, "flat", "circular")
	prevBtn.SetSensitive(false)
	prevBtn.SetTooltipText("Previous page")

	pageLbl := gtk.NewLabel(fmt.Sprintf("Page %d of %d", *currentPage, totalPages))
	pageLbl.AddCSSClass("caption")

	nextBtn := gtk.NewButtonFromIconName("go-next-symbolic")
	addCSSClasses(nextBtn, "flat", "circular")
	nextBtn.SetSensitive(totalPages > 1)
	nextBtn.SetTooltipText("Next page")

	updatePage := func(newPage int) {
		if newPage < 1 || newPage > totalPages {
			return
		}
		*currentPage = newPage
		pageLbl.SetText(fmt.Sprintf("Page %d of %d", *currentPage, totalPages))
		prevBtn.SetSensitive(*currentPage > 1)
		nextBtn.SetSensitive(*currentPage < totalPages)
		go func() {
			if tex := media.GetPDFPageTexture(filePath, newPage); tex != nil {
				glib.IdleAdd(func() {
					if *currentPage == newPage {
						setTexture(tex)
					}
				})
			}
		}()
	}

	prevBtn.ConnectClicked(func() { updatePage(*currentPage - 1) })
	nextBtn.ConnectClicked(func() { updatePage(*currentPage + 1) })

	switcherBox.Append(prevBtn)
	switcherBox.Append(pageLbl)
	switcherBox.Append(nextBtn)
	return switcherBox
}
