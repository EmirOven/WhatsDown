package ui

import (
	"context"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
)

func (ctrl *MainController) handleChatViewKeyPress(val uint, code uint, state gdk.ModifierType) bool {
	if ctrl.isAnySearchEntryFocused() {
		return false
	}

	if (state&gdk.ControlMask != 0 && (val == gdk.KEY_v || val == gdk.KEY_V)) ||
		(state&gdk.ShiftMask != 0 && val == gdk.KEY_Insert) {
		return ctrl.handlePasteShortcut()
	}

	if ctrl.currentChat == nil || ctrl.messageEntry == nil || ctrl.messageEntry.HasFocus() {
		return false
	}

	if state&(gdk.ControlMask|gdk.AltMask|gdk.SuperMask) != 0 {
		return false
	}

	ch := gdk.KeyvalToUnicode(val)
	if ch != 0 {
		ctrl.messageEntry.GrabFocus()
		if buf := ctrl.messageEntry.Buffer(); buf != nil {
			buf.InsertAtCursor(string(rune(ch)))
		}
		return true
	}

	return false
}

func (ctrl *MainController) isAnySearchEntryFocused() bool {
	return (ctrl.chatSearchEntry != nil && ctrl.chatSearchEntry.HasFocus()) ||
		(ctrl.chatsSearchEntry != nil && ctrl.chatsSearchEntry.HasFocus()) ||
		(ctrl.archivedSearchEntry != nil && ctrl.archivedSearchEntry.HasFocus()) ||
		(ctrl.callsSearchEntry != nil && ctrl.callsSearchEntry.HasFocus())
}

func (ctrl *MainController) handlePasteShortcut() bool {
	if ctrl.handleClipboardMediaPaste() {
		return true
	}
	if !ctrl.messageEntry.HasFocus() && ctrl.window != nil {
		clipboard := ctrl.window.Clipboard()
		if clipboard != nil {
			clipboard.ReadTextAsync(context.Background(), func(res gio.AsyncResulter) {
				txt, err := clipboard.ReadTextFinish(res)
				if err != nil || txt == "" {
					return
				}
				if targetPath, ok := parseFilePathFromText(txt); ok {
					glib.IdleAdd(func() {
						ctrl.showSendAttachmentDialog(targetPath)
					})
				} else {
					glib.IdleAdd(func() {
						ctrl.messageEntry.GrabFocus()
						ctrl.setMessageText(txt)
					})
				}
			})
			return true
		}
	}
	return false
}
