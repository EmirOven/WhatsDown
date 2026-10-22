package ui

import (
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

	chatKeyCtrl := gtk.NewEventControllerKey()
	chatKeyCtrl.SetPropagationPhase(gtk.PhaseCapture)
	chatKeyCtrl.ConnectKeyPressed(ctrl.handleChatViewKeyPress)
	if ctrl.window != nil {
		ctrl.window.AddController(chatKeyCtrl)
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
