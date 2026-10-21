package app

import (
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
)

// setQRTexture must be called on the GTK main thread.
func setQRTexture(texture *gdk.Texture) {
	qrTexture = texture
	if qrImage != nil {
		qrImage.SetFromPaintable(texture)
	}
}
