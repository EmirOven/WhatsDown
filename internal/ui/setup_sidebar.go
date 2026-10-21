package ui

import (
	"strings"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

func (ctrl *MainController) bindSidebarWidgets(
	builder *gtk.Builder,
	splitView *adw.NavigationSplitView,
	sidebarStack *gtk.Stack,
	chatsNavBtn *gtk.Button,
	archivedNavBtn *gtk.Button,
	chatsList *gtk.ListBox,
	archivedChatsList *gtk.ListBox,
) {
	ctrl.splitView = splitView
	ctrl.sidebarStack = sidebarStack
	ctrl.chatsNavBtn = chatsNavBtn
	ctrl.archivedNavBtn = archivedNavBtn
	ctrl.chatsList = chatsList
	ctrl.archivedChatsList = archivedChatsList

	ctrl.callsNavBtn, _ = castObject[*gtk.Button](builder, "calls_nav_btn")
	ctrl.callsPage, _ = castObject[*adw.ToolbarView](builder, "calls_page")
	ctrl.callsList, _ = castObject[*gtk.ListBox](builder, "calls_list")
	ctrl.callsSearchEntry, _ = castObject[*gtk.SearchEntry](builder, "calls_search_entry")
	ctrl.chatsSearchEntry, _ = castObject[*gtk.SearchEntry](builder, "chats_search_entry")
	ctrl.archivedSearchEntry, _ = castObject[*gtk.SearchEntry](builder, "archived_search_entry")

	chatsPage, _ := castObject[*adw.ToolbarView](builder, "chats_page")
	archivedPage, _ := castObject[*adw.ToolbarView](builder, "archived_page")
	settingsPage, _ := castObject[*adw.ToolbarView](builder, "settings_page")

	if sidebarStack != nil {
		if chatsPage != nil && sidebarStack.ChildByName("chats") == nil {
			sidebarStack.AddTitled(chatsPage, "chats", "Chats")
		}
		if ctrl.callsPage != nil && sidebarStack.ChildByName("calls") == nil {
			sidebarStack.AddTitled(ctrl.callsPage, "calls", "Calls")
		}
		if archivedPage != nil && sidebarStack.ChildByName("archived") == nil {
			sidebarStack.AddTitled(archivedPage, "archived", "Archived")
		}
		if settingsPage != nil && sidebarStack.ChildByName("settings") == nil {
			sidebarStack.AddTitled(settingsPage, "settings", "Settings")
		}
	}

	ctrl.bindSidebarNav()
}

func (ctrl *MainController) bindSidebarNav() {
	ctrl.sidebarStack.SetVisibleChildName("chats")
	ctrl.chatsNavBtn.AddCSSClass("active")

	ctrl.chatsNavBtn.ConnectClicked(func() {
		ctrl.switchSidebarView("chats")
		ctrl.loadChats()
	})
	if ctrl.callsNavBtn != nil {
		ctrl.callsNavBtn.ConnectClicked(func() {
			ctrl.switchSidebarView("calls")
			ctrl.loadCalls()
		})
	}
	ctrl.archivedNavBtn.ConnectClicked(func() {
		ctrl.switchSidebarView("archived")
		ctrl.loadArchivedChats()
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
