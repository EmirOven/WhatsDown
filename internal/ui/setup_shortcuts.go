package ui

import (
	"context"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

func (ctrl *MainController) bindShortcutsAndDrops(builder *gtk.Builder) {
	keyController := gtk.NewEventControllerKey()
	keyController.SetPropagationPhase(gtk.PhaseCapture)
	keyController.ConnectKeyPressed(ctrl.handleEntryKeyPress)
	ctrl.messageEntry.AddController(keyController)

	rightChatView, _ := castObject[*adw.ToolbarView](builder, "right_chat_view")
	if rightChatView != nil {
		chatKeyCtrl := gtk.NewEventControllerKey()
		chatKeyCtrl.SetPropagationPhase(gtk.PhaseCapture)
		chatKeyCtrl.ConnectKeyPressed(ctrl.handleChatViewKeyPress)
		rightChatView.AddController(chatKeyCtrl)
	}

	chatPage, _ := castObject[*gtk.Box](builder, "chat_page")
	if chatPage != nil {
		chatPage.AddController(ctrl.createFileDropTarget())
	} else if ctrl.messagesScrolledWindow != nil {
		ctrl.messagesScrolledWindow.AddController(ctrl.createFileDropTarget())
	}
	ctrl.messageEntry.AddController(ctrl.createFileDropTarget())
}

func (ctrl *MainController) createFileDropTarget() *gtk.DropTarget {
	target := gtk.NewDropTarget(gio.GTypeFile, gdk.ActionCopy)
	target.ConnectDrop(func(value *glib.Value, x, y float64) bool {
		if ctrl.currentChat == nil {
			return false
		}
		if filer, ok := value.GoValue().(gio.Filer); ok && filer != nil {
			if path := filer.Path(); path != "" {
				glib.IdleAdd(func() {
					ctrl.showSendAttachmentDialog(path)
				})
				return true
			}
		}
		return false
	})
	return target
}

func (ctrl *MainController) handleChatViewKeyPress(val uint, code uint, state gdk.ModifierType) bool {
	if ctrl.chatSearchEntry != nil && ctrl.chatSearchEntry.HasFocus() {
		return false
	}
	isCtrlV := (state&gdk.ControlMask != 0) && (val == gdk.KEY_v || val == gdk.KEY_V)
	isShiftInsert := (state&gdk.ShiftMask != 0) && (val == gdk.KEY_Insert)
	if isCtrlV || isShiftInsert {
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
	}
	return false
}
