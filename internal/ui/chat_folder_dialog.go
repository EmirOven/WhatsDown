package ui

import (
	"context"
	"fmt"
	"log"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/store"
)

func (c *MainController) showFolderError(err error) {
	log.Printf("chat folders: %v", err)
	if c.window == nil {
		return
	}
	alert := adw.NewAlertDialog("Could not update folder", err.Error())
	alert.AddResponse("close", "Close")
	alert.Present(c.window)
}

func (c *MainController) showChatFolderDialog(existing *store.ChatFolder, jids []string) {
	if c.store == nil {
		return
	}
	folder := store.ChatFolder{Color: "#3584e4"}
	title, action := "Create folder", "Create"
	if existing != nil {
		folder = *existing
		title, action = "Edit folder", "Save"
	}
	dialog := gtk.NewWindow()
	if c.window != nil {
		dialog.SetTransientFor(&c.window.Window)
	}
	dialog.SetTitle(title)
	dialog.SetModal(true)
	dialog.SetDefaultSize(380, 240)
	box := gtk.NewBox(gtk.OrientationVertical, 12)
	box.SetMarginTop(20)
	box.SetMarginBottom(20)
	box.SetMarginStart(20)
	box.SetMarginEnd(20)
	name := gtk.NewEntry()
	name.SetPlaceholderText("Folder name")
	name.SetActivatesDefault(true)
	name.SetText(folder.Name)
	box.Append(gtk.NewLabel("Name"))
	box.Append(name)
	colorRow := gtk.NewBox(gtk.OrientationHorizontal, 12)
	colorLabel := gtk.NewLabel("Colour")
	colorLabel.SetHExpand(true)
	colorLabel.SetXAlign(0)
	colorRow.Append(colorLabel)
	picker := gtk.NewColorDialog()
	picker.SetWithAlpha(false)
	color := gtk.NewColorDialogButton(picker)
	rgba := gdk.NewRGBA(0, 0, 0, 1)
	rgba.Parse(folder.Color)
	color.SetRGBA(&rgba)
	colorRow.Append(color)
	box.Append(colorRow)
	if existing == nil {
		description := gtk.NewLabel(fmt.Sprintf("Move %d selected chat(s) into this folder.", len(jids)))
		description.AddCSSClass("dim-label")
		box.Append(description)
	}
	errorLabel := gtk.NewLabel("")
	errorLabel.AddCSSClass("error")
	errorLabel.SetWrap(true)
	errorLabel.SetVisible(false)
	box.Append(errorLabel)
	actions := gtk.NewBox(gtk.OrientationHorizontal, 8)
	actions.SetHAlign(gtk.AlignEnd)
	cancel := gtk.NewButtonWithLabel("Cancel")
	save := gtk.NewButtonWithLabel(action)
	save.AddCSSClass("suggested-action")
	c.trackWidgetSignal(cancel, cancel, cancel.ConnectClicked(func() { dialog.Close() }))
	c.trackWidgetSignal(save, save, save.ConnectClicked(func() {
		value := color.RGBA()
		folder.Name = name.Text()
		folder.Color = fmt.Sprintf("#%02x%02x%02x", int(value.Red()*255+0.5), int(value.Green()*255+0.5), int(value.Blue()*255+0.5))
		var err error
		if existing == nil {
			_, err = c.store.CreateChatFolder(context.Background(), folder.Name, folder.Color, jids)
		} else {
			err = c.store.UpdateChatFolder(context.Background(), folder)
		}
		if err != nil {
			errorLabel.SetText(err.Error())
			errorLabel.SetVisible(true)
			return
		}
		dialog.Close()
		c.reloadFolderLists()
	}))
	actions.Append(cancel)
	actions.Append(save)
	box.Append(actions)
	dialog.SetChild(box)
	dialog.SetDefaultWidget(save)
	c.trackWidgetSignal(dialog, dialog, dialog.ConnectCloseRequest(func() bool { c.releaseMessageWidget(dialog); return false }))
	dialog.Present()
	name.GrabFocus()
}

func (c *MainController) attachFolderContextMenu(row *gtk.ListBoxRow, folder store.ChatFolder) {
	popover := gtk.NewPopover()
	popover.SetParent(row)
	popover.SetHasArrow(false)
	popover.AddCSSClass("popover-menu-box")
	box := gtk.NewBox(gtk.OrientationVertical, 4)
	edit := newMenuButtonContent("document-edit-symbolic", "Edit folder…")
	remove := newMenuButtonContent("user-trash-symbolic", "Delete folder")
	c.trackWidgetSignal(edit, edit, edit.ConnectClicked(func() { popover.Popdown(); c.showChatFolderDialog(&folder, nil) }))
	c.trackWidgetSignal(remove, remove, remove.ConnectClicked(func() {
		popover.Popdown()
		if err := c.store.DeleteChatFolder(context.Background(), folder.ID); err != nil {
			c.showFolderError(err)
			return
		}
		c.reloadFolderLists()
	}))
	box.Append(edit)
	box.Append(remove)
	popover.SetChild(box)
	gesture := gtk.NewGestureClick()
	gesture.SetButton(gdk.BUTTON_SECONDARY)
	c.trackWidgetSignal(row, gesture, gesture.ConnectPressed(func(_ int, x, y float64) {
		rect := gdk.NewRectangle(int(x), int(y), 1, 1)
		popover.SetPointingTo(&rect)
		popover.Popup()
	}))
	row.AddController(gesture)
}
