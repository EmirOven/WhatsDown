package ui

import (
	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

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

	if ctrl.archivedChatsList != nil {
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
	}

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

func setupRightHeaderBox(builder *gtk.Builder) {
	if hb, ok := castObject[*adw.HeaderBar](builder, "right_header_bar"); ok && hb != nil {
		if p := hb.Parent(); p != nil {
			addCSSClasses(p, "right-chat-header-box")
		}
	}
}
