package ui

import (
	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

func (ctrl *MainController) bindChatViewWidgets(
	builder *gtk.Builder,
	contentStack *gtk.Stack,
	activeChatBtn *gtk.Button,
	activeChatAvatar *adw.Avatar,
	activeChatTitle *adw.WindowTitle,
	messagesList *gtk.ListBox,
) {
	ctrl.contentStack = contentStack
	ctrl.activeChatBtn = activeChatBtn
	ctrl.activeChatAvatar = activeChatAvatar
	ctrl.activeChatTitle = activeChatTitle
	ctrl.messagesList = messagesList

	ctrl.chatSearchBtn, _ = castObject[*gtk.ToggleButton](builder, "chat_search_btn")
	ctrl.chatSearchBar, _ = castObject[*gtk.SearchBar](builder, "chat_search_bar")
	ctrl.chatSearchEntry, _ = castObject[*gtk.SearchEntry](builder, "chat_search_entry")
	ctrl.chatSearchCountLabel, _ = castObject[*gtk.Label](builder, "chat_search_count_label")
	ctrl.chatSearchPrevBtn, _ = castObject[*gtk.Button](builder, "chat_search_prev_btn")
	ctrl.chatSearchNextBtn, _ = castObject[*gtk.Button](builder, "chat_search_next_btn")
	ctrl.scrollToBottomBtn, _ = castObject[*gtk.Button](builder, "scroll_to_bottom_btn")
	ctrl.messagesScrolledWindow, _ = castObject[*gtk.ScrolledWindow](builder, "messages_scrolled_window")

	contentNavPage, _ := castObject[*adw.NavigationPage](builder, "content_nav_page")
	rightChatView, _ := castObject[*adw.ToolbarView](builder, "right_chat_view")
	if contentNavPage != nil && rightChatView != nil {
		contentNavPage.SetChild(rightChatView)
	}

	ctrl.activeChatBtn.SetSensitive(false)
	ctrl.activeChatBtn.ConnectClicked(func() {
		if ctrl.currentChat != nil {
			ctrl.showDetailsAndMedia(*ctrl.currentChat)
		}
	})

	ctrl.contentStack.SetVisibleChildName("empty")
	ctrl.bindChatSearch()
	ctrl.bindScrollAdjustment()
}

func (ctrl *MainController) bindChatSelection() {
	ctrl.chatsList.ConnectRowSelected(func(row *gtk.ListBoxRow) {
		if row == nil || ctrl.suppressChatSelection {
			return
		}
		idx := row.Index()
		if idx >= 0 && idx < len(ctrl.displayedChatRows) {
			item := ctrl.displayedChatRows[idx]
			if item.isCommunityHeader {
				ctrl.toggleCommunityExpanded(item.communityJID)
				return
			}
			target := item.chat
			if ctrl.currentChat != nil && (ctrl.currentChat.JID == target.JID || (ctrl.isSelfJID(ctrl.currentChat.JID) && ctrl.isSelfJID(target.JID))) {
				return
			}
			ctrl.selectChat(target)
		}
	})

	ctrl.archivedChatsList.ConnectRowSelected(func(row *gtk.ListBoxRow) {
		if row == nil || ctrl.suppressChatSelection {
			return
		}
		idx := row.Index()
		if idx >= 0 && idx < len(ctrl.displayedArchivedRows) {
			item := ctrl.displayedArchivedRows[idx]
			if item.isCommunityHeader {
				ctrl.toggleCommunityExpanded(item.communityJID)
				return
			}
			target := item.chat
			if ctrl.currentChat != nil && (ctrl.currentChat.JID == target.JID || (ctrl.isSelfJID(ctrl.currentChat.JID) && ctrl.isSelfJID(target.JID))) {
				return
			}
			ctrl.selectChat(target)
		}
	})

	if ctrl.callsList != nil {
		ctrl.callsList.ConnectRowSelected(func(row *gtk.ListBoxRow) {
			if row == nil {
				return
			}
			idx := row.Index()
			if idx >= 0 && idx < len(ctrl.calls) {
				ctrl.openCallChat(ctrl.calls[idx])
			}
		})
	}
}
