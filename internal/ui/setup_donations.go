package ui

import (
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/media"
)

const (
	liberapayURL    = "https://liberapay.com/postbeta"
	buyMeACoffeeURL = "https://buymeacoffee.com/postbeta"
)

func (ctrl *MainController) bindDonationButtons(builder *gtk.Builder) {
	if btn, ok := castObject[*gtk.Button](builder, "settings_donate_liberapay_btn"); ok && btn != nil {
		btn.ConnectClicked(func() {
			_ = media.OpenURL(liberapayURL)
		})
	}
	if btn, ok := castObject[*gtk.Button](builder, "settings_donate_bmac_btn"); ok && btn != nil {
		btn.ConnectClicked(func() {
			_ = media.OpenURL(buyMeACoffeeURL)
		})
	}
}
