package ui

import (
	"errors"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"go.mau.fi/whatsmeow"
	"postbetasoftware.com/whatsdown/internal/store"
)

// SetupMainWindow initializes the main application window and its controller.
func SetupMainWindow(builder *gtk.Builder, client *whatsmeow.Client, s *store.MessageStore) (*adw.ApplicationWindow, error) {
	window, ok1 := castObject[*adw.ApplicationWindow](builder, "main_window")
	splitView, ok2 := castObject[*adw.NavigationSplitView](builder, "split_view")
	chatsList, ok3 := castObject[*gtk.ListBox](builder, "chats_list")
	contentStack, ok4 := castObject[*gtk.Stack](builder, "content_stack")
	activeChatBtn, okBtn := castObject[*gtk.Button](builder, "active_chat_btn")
	activeChatAvatar, ok5 := castObject[*adw.Avatar](builder, "active_chat_avatar")
	activeChatTitle, ok6 := castObject[*adw.WindowTitle](builder, "active_chat_title")
	messagesList, ok7 := castObject[*gtk.ListBox](builder, "messages_list")
	messageEntry, ok8 := castObject[*gtk.TextView](builder, "message_entry")
	sendButton, ok9 := castObject[*gtk.Button](builder, "send_button")
	sidebarStack, ok12 := castObject[*adw.ViewStack](builder, "sidebar_stack")
	archivedChatsList, ok13 := castObject[*gtk.ListBox](builder, "archived_chats_list")

	if !ok1 || !ok2 || !ok3 || !ok4 || !okBtn || !ok5 || !ok6 || !ok7 || !ok8 || !ok9 || !ok12 || !ok13 {
		return nil, errors.New("WhatsDownMain.ui missing required widgets")
	}

	ctrl := &MainController{
		window:               window,
		client:               client,
		store:                s,
		expandedCommunities: make(map[string]bool),
		savedStickerHashes:   make(map[string]bool),
		autoScrollBottom:     true,
		messageStatusLabels:  make(map[string]*gtk.Label),
		onlineContacts:       make(map[string]bool),
		messageRowMap:        make(map[string]*gtk.ListBoxRow),
	}
	activeMainWindow = ctrl

	ctrl.bindSidebarWidgets(builder, splitView, sidebarStack, chatsList, archivedChatsList)
	ctrl.bindChatViewWidgets(builder, contentStack, activeChatBtn, activeChatAvatar, activeChatTitle, messagesList)
	ctrl.bindInputWidgets(builder, messageEntry, sendButton)
	ctrl.bindPopoverWidgets(builder)
	ctrl.bindSettingsWidgets(builder)
	ctrl.bindShortcutsAndDrops(builder)

	ctrl.setupVoiceRecording()
	ctrl.loadChats()
	ctrl.loadArchivedChats()
	ctrl.refreshSettingsInfo()

	return window, nil
}
