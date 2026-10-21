package main

import (
	_ "embed"

	"postbetasoftware.com/whatsdown/internal/ui"
)

//go:embed ui/WhatsDownInitial.ui
var initialPopupUI string

//go:embed ui/WhatsDownInitial.css
var initialPopupCSS string

//go:embed ui/WhatsDownSyncing.ui
var syncingUI string

//go:embed ui/WhatsDownSyncing.css
var syncingCSS string

//go:embed ui/WhatsDownMain.ui
var mainUI string

//go:embed ui/WhatsDownMain.css
var mainCSS string

//go:embed ui/components/ChatsMenu.ui
var chatsMenuUI string

//go:embed ui/components/CallsMenu.ui
var callsMenuUI string

//go:embed ui/components/ArchivedMenu.ui
var archivedMenuUI string

//go:embed ui/components/SettingsMenu.ui
var settingsMenuUI string

//go:embed ui/components/RightChatView.ui
var rightChatViewUI string

func init() {
	ui.SetEmbeddedAssets(ui.Assets{
		InitialPopupUI:  initialPopupUI,
		InitialPopupCSS: initialPopupCSS,
		SyncingUI:       syncingUI,
		SyncingCSS:      syncingCSS,
		MainUI:          mainUI,
		MainCSS:         mainCSS,
		ChatsMenuUI:     chatsMenuUI,
		CallsMenuUI:     callsMenuUI,
		ArchivedMenuUI:  archivedMenuUI,
		SettingsMenuUI:  settingsMenuUI,
		RightChatViewUI: rightChatViewUI,
	})
}
