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
	var css string
	switch currentPage {
	case pageSyncing:
		css = assets.SyncingCSS
	case pageMain:
		css = assets.MainCSS
	default:
		css = assets.InitialPopupCSS
	}

	if devMode {
		var path string
		switch currentPage {
		case pageSyncing:
			path = syncingCSSPath
		case pageMain:
			path = mainCSSPath
		default:
			path = initialCSSPath
		}
		if data, err := os.ReadFile(path); err == nil {
			css = string(data)
		}
	}

	if css != "" {
		provider.LoadFromData(css)
	} else {
		log.Printf("[CSS] no stylesheet found for page %v", currentPage)
	}
}
