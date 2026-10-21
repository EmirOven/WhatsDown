package ui

type Assets struct {
	InitialPopupUI  string
	InitialPopupCSS string
	SyncingUI       string
	SyncingCSS      string
	MainUI          string
	MainCSS         string
	ChatsMenuUI     string
	CallsMenuUI     string
	ArchivedMenuUI  string
	SettingsMenuUI  string
	RightChatViewUI string
}

var embeddedAssets Assets

func SetEmbeddedAssets(a Assets) {
	embeddedAssets = a
}

func GetAssets() Assets {
	return embeddedAssets
}
