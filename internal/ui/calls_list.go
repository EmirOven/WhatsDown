package ui

import (
	"context"
	"log"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

func (c *MainController) LoadCalls() {
	if c.store == nil || c.callsList == nil {
		return
	}
	query := ""
	if c.callsSearchEntry != nil {
		query = c.callsSearchEntry.Text()
	}
	calls, err := c.store.ListCalls(context.Background(), query)
	if err != nil {
		log.Printf("loading calls: %v", err)
		return
	}
	c.calls = calls
	c.callsList.RemoveAll()

	if len(calls) == 0 {
		emptyRow := gtk.NewListBoxRow()
		emptyRow.SetSelectable(false)
		emptyRow.SetActivatable(false)
		box := gtk.NewBox(gtk.OrientationVertical, 8)
		box.SetMarginTop(32)
		box.SetMarginBottom(32)
		box.SetHAlign(gtk.AlignCenter)

		icon := gtk.NewImageFromIconName("call-start-symbolic")
		icon.SetPixelSize(36)
		icon.AddCSSClass("dim-label")
		box.Append(icon)

		lbl := gtk.NewLabel("No calls yet")
		lbl.AddCSSClass("dim-label")
		box.Append(lbl)

		emptyRow.SetChild(box)
		c.callsList.Append(emptyRow)
		return
	}

	for _, call := range calls {
		c.callsList.Append(c.createCallRow(call))
	}
}

func (c *MainController) loadCalls() {
	c.LoadCalls()
}
