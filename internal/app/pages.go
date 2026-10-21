package app

import "postbetasoftware.com/whatsdown/internal/ui"

const (
	uiDir          = "ui"
	initialUIPath  = "ui/WhatsDownInitial.ui"
	initialCSSPath = "ui/WhatsDownInitial.css"
	syncingUIPath  = "ui/WhatsDownSyncing.ui"
	syncingCSSPath = "ui/WhatsDownSyncing.css"
	mainUIPath     = "ui/WhatsDownMain.ui"
	mainCSSPath    = "ui/WhatsDownMain.css"
)

func (p page) uiPath() string {
	switch p {
	case pageSyncing:
		return syncingUIPath
	case pageMain:
		return mainUIPath
	default:
		return initialUIPath
	}
}

func (p page) embeddedUI() string {
	assets := ui.GetAssets()
	switch p {
	case pageSyncing:
		return assets.SyncingUI
	case pageMain:
		return assets.MainUI
	default:
		return assets.InitialPopupUI
	}
}
