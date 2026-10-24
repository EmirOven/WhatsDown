package ui

import (
	"context"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

func (c *MainController) persistFolderCollapsed(id string, collapsed bool) {
	if c.folderCollapsedOverrides == nil {
		c.folderCollapsedOverrides = make(map[string]bool)
	}
	c.folderCollapsedOverrides[id] = collapsed
	c.folderStateMu.Lock()
	if c.folderStatePending == nil {
		c.folderStatePending = make(map[string]bool)
	}
	c.folderStatePending[id] = collapsed
	if c.folderStateWriting {
		c.folderStateMu.Unlock()
		return
	}
	c.folderStateWriting = true
	c.folderStateMu.Unlock()
	go func() {
		for {
			c.folderStateMu.Lock()
			pending := c.folderStatePending
			c.folderStatePending = nil
			if len(pending) == 0 {
				c.folderStateWriting = false
				c.folderStateMu.Unlock()
				return
			}
			c.folderStateMu.Unlock()
			for id, value := range pending {
				err := c.store.SetChatFolderCollapsed(context.Background(), id, value)
				glib.IdleAdd(func() {
					if current, ok := c.folderCollapsedOverrides[id]; ok && current == value {
						delete(c.folderCollapsedOverrides, id)
					}
					if err != nil {
						c.showFolderError(err)
					}
				})
			}
		}
	}()
}

func (c *MainController) setFolderRowsExpanded(list *gtk.ListBox, rows []chatRowItem, id string, expanded bool) {
	if list == nil {
		return
	}
	var adjustment *gtk.Adjustment
	var position float64
	for parent := list.Parent(); parent != nil; parent = gtk.BaseWidget(parent).Parent() {
		if scroll, ok := parent.(*gtk.ScrolledWindow); ok {
			adjustment = scroll.VAdjustment()
			position = adjustment.Value()
			break
		}
	}
	var header *gtk.ListBoxRow
	for i, item := range rows {
		if item.folderID != id {
			continue
		}
		row := list.RowAtIndex(i)
		if row == nil {
			continue
		}
		if item.isFolderHeader {
			header = row
			if box, ok := row.Child().(*gtk.Box); ok {
				if arrow, ok := box.LastChild().(*gtk.Image); ok {
					icon := "pan-end-symbolic"
					if expanded {
						icon = "pan-down-symbolic"
					}
					arrow.SetFromIconName(icon)
				}
			}
		} else {
			row.SetVisible(expanded)
		}
	}
	if header != nil {
		header.GrabFocus()
	}
	if adjustment != nil {
		adjustment.SetValue(position)
		glib.IdleAdd(func() { adjustment.SetValue(position) })
	}
}
