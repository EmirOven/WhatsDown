package ui

import (
	"context"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/store"
)

func (c *MainController) openCallChat(call store.CallRecord) {
	chatJID := call.ChatJID
	if chatJID == "" {
		chatJID = call.CallerJID
	}
	targetChat, err := c.store.GetChatOrContact(context.Background(), chatJID)
	if err != nil || targetChat.JID == "" {
		targetChat = store.Chat{
			JID:  chatJID,
			Name: call.CallerName,
		}
	}
	c.selectChat(targetChat)
	c.switchSidebarView("chats")
}

func (c *MainController) attachCallContextMenu(row *gtk.ListBoxRow, call store.CallRecord) {
	popover := gtk.NewPopover()
	popover.SetParent(row)
	popover.SetHasArrow(false)
	popover.AddCSSClass("popover-menu-box")

	menuBox := gtk.NewBox(gtk.OrientationVertical, 2)
	menuBox.SetMarginTop(4)
	menuBox.SetMarginBottom(4)
	menuBox.SetMarginStart(4)
	menuBox.SetMarginEnd(4)

	openBtn := gtk.NewButton()
	openBtn.AddCSSClass("flat")
	openBtn.AddCSSClass("popover-menu-btn")
	openBtnBox := gtk.NewBox(gtk.OrientationHorizontal, 10)
	openBtnBox.Append(gtk.NewImageFromIconName("chat-message-new-symbolic"))
	openBtnBox.Append(gtk.NewLabel("Open Chat"))
	openBtn.SetChild(openBtnBox)
	openBtn.ConnectClicked(func() {
		popover.Popdown()
		c.openCallChat(call)
	})
	menuBox.Append(openBtn)

	delBtn := gtk.NewButton()
	delBtn.AddCSSClass("flat")
	delBtn.AddCSSClass("popover-menu-btn")
	delBtnBox := gtk.NewBox(gtk.OrientationHorizontal, 10)
	delBtnBox.Append(gtk.NewImageFromIconName("user-trash-symbolic"))
	delBtnBox.Append(gtk.NewLabel("Delete from History"))
	delBtn.SetChild(delBtnBox)
	delBtn.ConnectClicked(func() {
		popover.Popdown()
		if c.store != nil {
			go func() {
				_ = c.store.DeleteCall(context.Background(), call.ID)
				glib.IdleAdd(func() {
					c.loadCalls()
				})
			}()
		}
	})
	menuBox.Append(delBtn)
	popover.SetChild(menuBox)

	gesture := gtk.NewGestureClick()
	gesture.SetButton(3)
	gesture.ConnectPressed(func(n int, x, y float64) {
		rect := gdk.NewRectangle(int(x), int(y), 1, 1)
		popover.SetPointingTo(&rect)
		popover.Popup()
	})
	row.AddController(gesture)
}
