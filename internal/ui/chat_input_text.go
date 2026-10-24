package ui

import (
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

func (c *MainController) getMessageText() string {
	if c.messageEntry == nil {
		return ""
	}
	buf := c.messageEntry.Buffer()
	if buf == nil {
		return ""
	}
	return buf.Text(buf.StartIter(), buf.EndIter(), true)
}

func (c *MainController) setMessageText(text string) {
	if c.messageEntry == nil {
		return
	}
	buf := c.messageEntry.Buffer()
	if buf == nil {
		return
	}
	buf.SetText(text)
	buf.PlaceCursor(buf.EndIter())
	if c.messagePlaceholder != nil {
		c.messagePlaceholder.SetVisible(text == "")
	}
	c.updateMessageEntryHeight()
}

func (c *MainController) clearMessageText() {
	c.setMessageText("")
	c.hideEmojiSuggestions()
}

func (c *MainController) updateMessageEntryHeight() {
	if c.messageEntry == nil || c.messageEntryScrolled == nil {
		return
	}
	if c.inputHeightPending {
		return
	}
	c.inputHeightPending = true
	glib.IdleAdd(func() {
		c.inputHeightPending = false
		if c.messageEntry == nil || c.messageEntryScrolled == nil {
			return
		}
		buf := c.messageEntry.Buffer()
		if buf == nil {
			return
		}
		endIter := buf.EndIter()
		yEnd, hEnd := c.messageEntry.LineYrange(endIter)
		if hEnd <= 0 {
			hEnd = 18
		}
		contentH := yEnd + hEnd + c.messageEntry.TopMargin() + c.messageEntry.BottomMargin() + 4

		singleLineH := 34
		max6LinesH := 126

		targetH := contentH
		if targetH < singleLineH {
			targetH = singleLineH
		}
		if targetH > max6LinesH {
			targetH = max6LinesH

		}
		if targetH != c.inputHeight {
			c.inputHeight = targetH
			policy := gtk.PolicyNever
			if contentH > max6LinesH {
				policy = gtk.PolicyAutomatic
			}
			c.messageEntryScrolled.SetPolicy(gtk.PolicyNever, policy)
			c.messageEntryScrolled.SetSizeRequest(-1, targetH)
		}
		if contentH > max6LinesH {
			c.messageEntry.ScrollToMark(buf.GetInsert(), 0, false, 0, 0)
		}
	})
}
