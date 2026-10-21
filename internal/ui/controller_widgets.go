package ui

import (
	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/audio"
)

type sidebarWidgets struct {
	splitView           *adw.NavigationSplitView
	sidebarStack        *gtk.Stack
	chatsNavBtn         *gtk.Button
	callsNavBtn         *gtk.Button
	archivedNavBtn      *gtk.Button
	settingsNavBtn      *gtk.Button
	chatsSearchEntry    *gtk.SearchEntry
	chatsList           *gtk.ListBox
	callsPage           *adw.ToolbarView
	callsList           *gtk.ListBox
	callsSearchEntry    *gtk.SearchEntry
	archivedSearchEntry *gtk.SearchEntry
	archivedChatsList   *gtk.ListBox
}

type chatViewWidgets struct {
	contentStack           *gtk.Stack
	activeChatBtn          *gtk.Button
	activeChatAvatar       *adw.Avatar
	activeChatTitle        *adw.WindowTitle
	chatSearchBtn          *gtk.ToggleButton
	chatSearchBar          *gtk.SearchBar
	chatSearchEntry        *gtk.SearchEntry
	chatSearchCountLabel   *gtk.Label
	chatSearchPrevBtn      *gtk.Button
	chatSearchNextBtn      *gtk.Button
	scrollToBottomBtn      *gtk.Button
	messagesScrolledWindow *gtk.ScrolledWindow
	messagesList           *gtk.ListBox
}

type inputWidgets struct {
	messageEntry          *gtk.TextView
	messageEntryScrolled  *gtk.ScrolledWindow
	messagePlaceholder    *gtk.Label
	sendButton            *gtk.Button
	attachButton          *gtk.MenuButton
	attachPopover         *gtk.Popover
	attachPhotoBtn        *gtk.Button
	attachDocBtn          *gtk.Button
	attachPollBtn         *gtk.Button
	emojiButton           *gtk.MenuButton
	emojiChooser          *gtk.EmojiChooser
	stickerButton         *gtk.MenuButton
	stickerPopover        *gtk.Popover
	stickerFlowbox        *gtk.FlowBox
	stickerEmptyBox       *gtk.Box
	stickerScrolledWindow *gtk.ScrolledWindow
	addStickerBtn         *gtk.Button
}

type voiceWidgets struct {
	chatInputStack   *gtk.Stack
	voiceRecordBtn   *gtk.Button
	voiceDeleteBtn   *gtk.Button
	voiceStatusIcon  *gtk.Image
	voiceTimerLabel  *gtk.Label
	voiceMeterArea   *gtk.DrawingArea
	voicePauseBtn    *gtk.Button
	voiceStopBtn     *gtk.Button
	voicePlayBtn     *gtk.Button
	voiceSendBtn     *gtk.Button
	voiceRecorder    *audio.Recorder
	voiceTimerHandle glib.SourceHandle
	voicePreviewPath string
	voiceIsPlaying   bool
}

type settingsWidgets struct {
	settingsProfileRow    *adw.ActionRow
	settingsProfileAvatar *adw.Avatar
	settingsResyncBtn     *gtk.Button
	settingsDbPathRow     *adw.ActionRow
	settingsStatsRow      *adw.ActionRow
	settingsClearCacheBtn *gtk.Button
	settingsThemeRow      *adw.ComboRow
	settingsLogoutBtn     *gtk.Button
}
