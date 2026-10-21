package ui

import (
	"fmt"
	"strings"

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

	inBox := gtk.NewBox(gtk.OrientationHorizontal, 0)
	inBox.AddCSSClass("wallpaper-preview-incoming")
	inBox.SetHAlign(gtk.AlignStart)
	inBox.Append(gtk.NewLabel("Hey there! How are you?"))
	previewBox.Append(inBox)

	outBox := gtk.NewBox(gtk.OrientationHorizontal, 0)
	outBox.AddCSSClass("wallpaper-preview-outgoing")
	outBox.SetHAlign(gtk.AlignEnd)
	outBox.Append(gtk.NewLabel("Doing great, thanks!"))
	previewBox.Append(outBox)

	pw := &wallpaperPreviewWidgets{container: previewBox, cssProvider: provider}
	return previewBox, pw
}

func (pw *wallpaperPreviewWidgets) update(wp store.ChatWallpaper) {
	if pw == nil || pw.cssProvider == nil {
		return
	}

	var bgStyle string
	var isLight bool

	switch wp.Kind {
	case "color":
		if wp.Value != "" {
			bgStyle = fmt.Sprintf("background-color: %s; background-image: none;", wp.Value)
			if rgba, ok := parseHexColor(wp.Value); ok {
				isLight = isColorLight(int(rgba.Red()*255), int(rgba.Green()*255), int(rgba.Blue()*255))
			}
		}
	case "image":
		if wp.Value != "" {
			urlVal := wp.Value
			if !strings.HasPrefix(urlVal, "file://") {
				urlVal = "file://" + urlVal
			}
			bgStyle = fmt.Sprintf("background-image: url(\"%s\"); background-size: cover; background-position: center; background-repeat: no-repeat;", urlVal)
			if r, g, b, ok := computeAverageImageColor(wp.Value); ok {
				isLight = isColorLight(r, g, b)
			}
		}
	}

	pw.applyStyle(bgStyle, isLight)
}

func (pw *wallpaperPreviewWidgets) applyStyle(bgStyle string, isLight bool) {
	if bgStyle == "" {
		pw.cssProvider.LoadFromData(`
			.wallpaper-preview-box { background: none; }
			.wallpaper-preview-incoming { background-color: alpha(currentColor, 0.08); color: currentColor; }
			.wallpaper-preview-incoming label { color: currentColor; }
		`)
		return
	}

	var bubbleBg, bubbleFg string
	if isLight {
		bubbleBg, bubbleFg = "rgba(0, 0, 0, 0.72)", "#ffffff"
	} else {
		bubbleBg, bubbleFg = "rgba(255, 255, 255, 0.88)", "#000000"
	}

	pw.cssProvider.LoadFromData(fmt.Sprintf(`
		.wallpaper-preview-box { %s }
		.wallpaper-preview-incoming { background-color: %s !important; color: %s !important; }
		.wallpaper-preview-incoming label { color: %s !important; }
	`, bgStyle, bubbleBg, bubbleFg, bubbleFg))
}
