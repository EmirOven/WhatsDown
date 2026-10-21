package ui

import (
	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"go.mau.fi/whatsmeow"
	"postbetasoftware.com/whatsdown/internal/store"
)

// MainController manages state and UI interactions of the WhatsDownMain window.
type MainController struct {
	window *adw.ApplicationWindow
	client *whatsmeow.Client
	store  *store.MessageStore

	sidebarWidgets
	chatViewWidgets
	inputWidgets
	voiceWidgets
	settingsWidgets

	savedStickers         []store.SavedSticker
	savedStickerHashes    map[string]bool
	suppressChatSelection bool
	autoScrollBottom      bool
	isAutoScrolling       bool
	messageStatusLabels   map[string]*gtk.Label

	// Reply preview widgets & state
	replyPreviewBar    *gtk.Box
	replyPreviewSender *gtk.Label
	replyPreviewText   *gtk.Label
	cancelReplyBtn     *gtk.Button
	replyingTo         *store.StoredMessage
	messageRowMap      map[string]*gtk.ListBoxRow

	// Markdown preview widgets
	markdownPreviewBar   *gtk.Box
	markdownPreviewLabel *gtk.Label

	// Emoji colon autocomplete widgets & state
	emojiSuggestionBar   *gtk.Box
	emojiSuggestions     []string
	emojiSuggestionIndex int
	emojiColonStartPos   int

	// Edit message widgets & state
	editBannerBar  *gtk.Box
	editBannerText *gtk.Label
	cancelEditBtn  *gtk.Button
	editingMessage *store.StoredMessage

	// Typing indicator widgets & state
	typingIndicatorBar  *gtk.Box
	typingDot1          *gtk.Label
	typingDot2          *gtk.Label
	typingDot3          *gtk.Label
	typingTimeoutSource glib.SourceHandle
	typingAnimSource    glib.SourceHandle
	onlineContacts      map[string]bool

	currentChat             *store.Chat
	chats                   []store.Chat
	archivedChats           []store.Chat
	calls                   []store.CallRecord
	expandedCommunities    map[string]bool
	displayedChatRows       []chatRowItem
	displayedArchivedRows   []chatRowItem
	currentMessages         []store.StoredMessage
	chatsSearchFilter       string
	archivedSearchFilter    string
	messagesSearchFilter    string
	searchMatchIndices      []int
	currentSearchMatchIndex int
}
var activeMainWindow *MainController

func ActiveMainWindow() *MainController {
	return activeMainWindow
}

func SetActiveMainWindow(c *MainController) {
	activeMainWindow = c
}

func (c *MainController) CurrentChat() *store.Chat {
	if c == nil {
		return nil
	}
	return c.currentChat
}

func (c *MainController) IsSameChat(a, b string) bool {
	if c == nil {
		return false
	}
	return c.isSameChat(a, b)
}
