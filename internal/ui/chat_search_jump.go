package ui

import (
	"github.com/diamondburned/gotk4/pkg/glib/v2"
)

func (c *MainController) jumpToSearchMatch(index int) {
	if index < 0 || index >= len(c.searchMatchIndices) {
		return
	}
	msgIdx := c.searchMatchIndices[index]
	row := c.messagesList.RowAtIndex(msgIdx)
	if row != nil {
		row.AddCSSClass("message-highlighted")
		glib.TimeoutAdd(2000, func() bool {
			row.RemoveCSSClass("message-highlighted")
			return false
		})

		if c.messagesScrolledWindow != nil {
			if _, destY, ok := row.TranslateCoordinates(c.messagesList, 0, 0); ok {
				if adj := c.messagesScrolledWindow.VAdjustment(); adj != nil {
					target := destY - adj.PageSize()/3
					if target < 0 {
						target = 0
					}
					maxVal := adj.Upper() - adj.PageSize()
					if maxVal > 0 && target > maxVal {
						target = maxVal
					}
					adj.SetValue(target)
				}
			}
		}
	}
}
