package ui

import (
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
)

func (ctrl *MainController) handleEntryKeyPress(val uint, code uint, state gdk.ModifierType) bool {
	if ctrl.isMentionPopupVisible() {
		if val == gdk.KEY_Up {
			ctrl.navigateMentionPopup(-1)
			return true
		}
		if val == gdk.KEY_Down {
			ctrl.navigateMentionPopup(1)
			return true
		}
		if val == gdk.KEY_Return || val == gdk.KEY_KP_Enter || val == gdk.KEY_ISO_Enter || val == gdk.KEY_Tab {
			ctrl.commitSelectedMention()
			return true
		}
		if val == gdk.KEY_Escape {
			ctrl.hideMentionPopup()
			return true
		}
	}

	if ctrl.isEmojiSuggestionVisible() {
		if val == gdk.KEY_Left {
			ctrl.navigateEmojiSuggestion(-1)
			return true
		}
		if val == gdk.KEY_Right {
			ctrl.navigateEmojiSuggestion(1)
			return true
		}
		if val == gdk.KEY_Return || val == gdk.KEY_KP_Enter || val == gdk.KEY_ISO_Enter {
			ctrl.commitSelectedEmoji()
			return true
		}
		if val == gdk.KEY_Escape {
			ctrl.hideEmojiSuggestions()
			return true
		}
	}

	if val == gdk.KEY_Escape {
		if ctrl.editingMessage != nil {
			ctrl.cancelEditing()
			return true
		}
		if ctrl.replyingTo != nil {
			ctrl.cancelReply()
			return true
		}
	}

	if val == gdk.KEY_Return || val == gdk.KEY_KP_Enter || val == gdk.KEY_ISO_Enter {
		isCtrl := state&gdk.ControlMask != 0
		isShift := state&gdk.ShiftMask != 0
		if isCtrl || isShift {
			if ctrl.messageEntry != nil {
				if buf := ctrl.messageEntry.Buffer(); buf != nil {
					buf.InsertAtCursor("\n")
				}
			}
			return true
		}
		ctrl.sendMessage()
		return true
	}

	isCtrlV := (state&gdk.ControlMask != 0) && (val == gdk.KEY_v || val == gdk.KEY_V)
	isShiftInsert := (state&gdk.ShiftMask != 0) && (val == gdk.KEY_Insert)
	if isCtrlV || isShiftInsert {
		if ctrl.handleClipboardMediaPaste() {
			return true
		}
	}
	return false
}
