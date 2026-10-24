package ui

import (
	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

func (ctrl *MainController) bindChatSelection() {
	ctrl.bindChatListSelection(ctrl.chatsList, false)
	ctrl.bindChatListSelection(ctrl.archivedChatsList, true)

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
