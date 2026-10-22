package ui

import (
	"fmt"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/store"
)

type wallpaperPreviewWidgets struct {
	container   *gtk.Box
	cssProvider *gtk.CSSProvider
}

func (c *MainController) buildWallpaperPreview() (*gtk.Box, *wallpaperPreviewWidgets) {
	previewBox := gtk.NewBox(gtk.OrientationVertical, 8)
	previewBox.AddCSSClass("wallpaper-preview-box")
	previewBox.SetHExpand(true)

	provider := gtk.NewCSSProvider()
	previewBox.StyleContext().AddProvider(provider, gtk.STYLE_PROVIDER_PRIORITY_APPLICATION+15)

	inRow := createPreviewBubble("wallpaper-preview-incoming", gtk.AlignStart, "Hey there! How are you?", "10:42", "")
	inRow.StyleContext().AddProvider(provider, gtk.STYLE_PROVIDER_PRIORITY_APPLICATION+15)
	previewBox.Append(inRow)

	outRow := createPreviewBubble("wallpaper-preview-outgoing", gtk.AlignEnd, "Doing great, thanks!", "10:43", "✓✓")
	previewBox.Append(outRow)

	return previewBox, &wallpaperPreviewWidgets{container: previewBox, cssProvider: provider}
}

func createPreviewBubble(cssClass string, align gtk.Align, text, timeStr, statusStr string) *gtk.Box {
	box := gtk.NewBox(gtk.OrientationVertical, 2)
	addCSSClasses(box, "wallpaper-preview-bubble", cssClass)
	box.SetHAlign(align)
	box.Append(gtk.NewLabel(text))

	footer := gtk.NewBox(gtk.OrientationHorizontal, 4)
	footer.SetHAlign(gtk.AlignEnd)
	timeLbl := gtk.NewLabel(timeStr)
	timeLbl.AddCSSClass("wallpaper-preview-time")
	footer.Append(timeLbl)

	if statusStr != "" {
		stLbl := gtk.NewLabel(statusStr)
		stLbl.AddCSSClass("wallpaper-preview-status")
		footer.Append(stLbl)
	}
	box.Append(footer)
	return box
}

func (pw *wallpaperPreviewWidgets) update(wp store.ChatWallpaper) {
	if pw == nil || pw.cssProvider == nil {
		return
	}
	bgStyle, bubbleBg, bubbleFg := extractWallpaperStyle(wp)
	if bgStyle == "" {
		pw.cssProvider.LoadFromData(`
			.wallpaper-preview-box { background: none; }
			.wallpaper-preview-incoming { background-color: alpha(currentColor, 0.08); color: currentColor; }
			.wallpaper-preview-incoming label { color: currentColor; }
		`)
		return
	}

	pw.cssProvider.LoadFromData(fmt.Sprintf(`
		.wallpaper-preview-box { %s }
		.wallpaper-preview-incoming { background-color: %s; color: %s; }
		.wallpaper-preview-incoming label { color: %s; }
	`, bgStyle, bubbleBg, bubbleFg, bubbleFg))
}
