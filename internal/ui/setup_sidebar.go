package ui

import (
	"strings"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

func (ctrl *MainController) bindSidebarWidgets(
	builder *gtk.Builder,
	splitView *adw.NavigationSplitView,
	sidebarStack *adw.ViewStack,
	chatsList *gtk.ListBox,
	archivedChatsList *gtk.ListBox,
) {
	ctrl.splitView = splitView
	ctrl.sidebarStack = sidebarStack
	ctrl.chatsList = chatsList
	ctrl.markAllReadBtn, _ = castObject[*gtk.Button](builder, "mark_all_read_btn")
	if ctrl.markAllReadBtn != nil {
		ctrl.markAllReadBtn.ConnectClicked(ctrl.markAllChatsRead)
	}
	ctrl.archivedChatsList = archivedChatsList

	ctrl.callsPage, _ = castObject[*adw.ToolbarView](builder, "calls_page")
	ctrl.callsList, _ = castObject[*gtk.ListBox](builder, "calls_list")
	ctrl.callsSearchEntry, _ = castObject[*gtk.SearchEntry](builder, "calls_search_entry")
	ctrl.chatsSearchEntry, _ = castObject[*gtk.SearchEntry](builder, "chats_search_entry")
	ctrl.archivedSearchEntry, _ = castObject[*gtk.SearchEntry](builder, "archived_search_entry")

	if newGroupBtn, ok := castObject[*gtk.Button](builder, "new_group_btn"); ok && newGroupBtn != nil {
		newGroupBtn.ConnectClicked(func() {
			ctrl.showCreateGroupDialog()
		})
	}

	chatsPage, _ := castObject[*adw.ToolbarView](builder, "chats_page")
	archivedPage, _ := castObject[*adw.ToolbarView](builder, "archived_page")
	settingsPage, _ := castObject[*adw.ToolbarView](builder, "settings_page")

	if sidebarStack != nil {
		if chatsPage != nil && sidebarStack.ChildByName("chats") == nil {
			sidebarStack.AddTitledWithIcon(chatsPage, "chats", "Chats", "chat-message-new-symbolic")
		}
		if ctrl.callsPage != nil && sidebarStack.ChildByName("calls") == nil {
			sidebarStack.AddTitledWithIcon(ctrl.callsPage, "calls", "Calls", "call-start-symbolic")
		}
		if archivedPage != nil && sidebarStack.ChildByName("archived") == nil {
			sidebarStack.AddTitledWithIcon(archivedPage, "archived", "Archived", "folder-download-symbolic")
		}
		if settingsPage != nil && sidebarStack.ChildByName("settings") == nil {
			sidebarStack.AddTitledWithIcon(settingsPage, "settings", "Settings", "preferences-system-symbolic")
		}
	}

	ctrl.bindSidebarNav()
}

func (ctrl *MainController) bindSidebarNav() {
	ctrl.sidebarStack.SetVisibleChildName("chats")
	ctrl.sidebarStack.NotifyProperty("visible-child-name", func() {
		switch ctrl.sidebarStack.VisibleChildName() {
		case "chats":
			ctrl.loadChats()
		case "calls":
			ctrl.loadCalls()
		case "archived":
			ctrl.loadArchivedChats()
		case "settings":
			ctrl.refreshSettingsInfo()
		}
	})

	if ctrl.chatsSearchEntry != nil {
		ctrl.chatsSearchEntry.ConnectSearchChanged(func() {
			ctrl.chatsSearchFilter = strings.TrimSpace(ctrl.chatsSearchEntry.Text())
			ctrl.loadChats()
		})
	}
	if ctrl.archivedSearchEntry != nil {
		ctrl.archivedSearchEntry.ConnectSearchChanged(func() {
			ctrl.archivedSearchFilter = strings.TrimSpace(ctrl.archivedSearchEntry.Text())
			ctrl.loadArchivedChats()
		})
	}
	if ctrl.callsSearchEntry != nil {
		ctrl.callsSearchEntry.ConnectSearchChanged(func() {
			ctrl.loadCalls()
		})
	}

	ctrl.bindChatSelection()
}
