package app

import (
	"log"
	"os"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/ui"
)

func newCSSProvider() *gtk.CSSProvider {
	provider := gtk.NewCSSProvider()
	if display := gdk.DisplayGetDefault(); display != nil {
		gtk.StyleContextAddProviderForDisplay(display, provider, gtk.STYLE_PROVIDER_PRIORITY_APPLICATION)
	}
	return provider
}

func loadCSS(provider *gtk.CSSProvider) {
	assets := ui.GetAssets()
	css := assets.InitialPopupCSS + "\n" + assets.SyncingCSS + "\n" + assets.MainCSS

	if devMode {
		css = ""
		for _, path := range []string{initialCSSPath, syncingCSSPath, mainCSSPath} {
			if data, err := os.ReadFile(path); err == nil {
				css += string(data) + "\n"
			} else {
				log.Printf("[CSS] reading %s: %v", path, err)
			}
		}
	}

	if css != "" {
		provider.LoadFromData(css)
	} else {
		log.Printf("[CSS] no stylesheets loaded")
	}
}
