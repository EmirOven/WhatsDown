package ui

import (
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

func (c *MainController) scrollToBottom() {
	if c.messagesScrolledWindow == nil {
		return
	}
	if vp, ok := c.messagesScrolledWindow.Child().(*gtk.Viewport); ok && c.messagesList != nil {
		cnt := len(c.currentMessages)
		if cnt > 0 {
			if lastRow := c.messagesList.RowAtIndex(cnt - 1); lastRow != nil {
				vp.ScrollTo(lastRow, nil)
			}
		}
	}
	adj := c.messagesScrolledWindow.VAdjustment()
	if adj != nil {
		maxVal := adj.Upper() - adj.PageSize()
		if maxVal > 0 {
			adj.SetValue(maxVal)
		} else {
			adj.SetValue(0)
		}
	}
	if c.scrollToBottomBtn != nil {
		c.scrollToBottomBtn.SetVisible(false)
	}
}

func (c *MainController) scrollToBottomAnimated() {
	c.autoScrollBottom = true
	c.isAutoScrolling = true
	c.scrollToBottom()
	glib.IdleAdd(func() {
		if c.autoScrollBottom {
			c.scrollToBottom()
		}
		c.isAutoScrolling = false
	})
}

func (c *MainController) scrollToMessageID(id string) {
	if row, ok := c.messageRowMap[id]; ok && row != nil {
		row.GrabFocus()
		if vp, ok := c.messagesScrolledWindow.Child().(*gtk.Viewport); ok {
			vp.ScrollTo(row, nil)
		}
		row.AddCSSClass("message-highlighted")
		glib.TimeoutAdd(2000, func() bool {
			row.RemoveCSSClass("message-highlighted")
			return false
		})
	}
}
