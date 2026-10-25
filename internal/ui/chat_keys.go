package ui

import (
	"context"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

func (ctrl *MainController) handleChatViewKeyPress(val uint, code uint, state gdk.ModifierType) bool {
	if ctrl.window != nil && ctrl.window.VisibleDialog() != nil {
		return false
	}
	if state&gdk.ControlMask != 0 && (val == gdk.KEY_slash || val == gdk.KEY_question) {
		return ctrl.handleAppShortcut(val, state)
	}
	if ctrl.mediaViewer != nil {
		if val == gdk.KEY_Escape {
			ctrl.mediaViewer.Close()
			return true
		}
		return false
	}
	if ctrl.handleAppShortcut(val, state) {
		return true
	}
	if ctrl.messageSelection != nil && ctrl.messageSelection.active {
		if val == gdk.KEY_Escape {
			ctrl.cancelMessageSelection()
			return true
		}
		if !ctrl.isAnySearchEntryFocused() && state&gdk.ControlMask != 0 && (val == gdk.KEY_a || val == gdk.KEY_A) {
			for _, msg := range ctrl.currentMessages {
				ctrl.messageSelection.selected[msg.ID] = true
			}
			ctrl.updateMessageSelectionUI()
			return true
		}
		return false
	}
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
	if ctrl.window == nil {
		return false
	}
	focus := ctrl.window.Focus()
	if focus == nil {
		return false
	}
	w := gtk.BaseWidget(focus)
	for _, entry := range []*gtk.SearchEntry{ctrl.chatSearchEntry, ctrl.chatsSearchEntry, ctrl.archivedSearchEntry, ctrl.callsSearchEntry} {
		if entry != nil && (entry.HasFocus() || w.IsAncestor(entry)) {
			return true
		}
	}
	return false
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
