package ui

import (
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

func (c *MainController) setTypingIndicator(isTyping bool) {
	if c.typingIndicatorBar == nil {
		return
	}

	if c.typingTimeoutSource != 0 {
		glib.SourceRemove(c.typingTimeoutSource)
		c.typingTimeoutSource = 0
	}

	if !isTyping {
		c.typingIndicatorBar.SetVisible(false)
		if c.typingAnimSource != 0 {
			glib.SourceRemove(c.typingAnimSource)
			c.typingAnimSource = 0
		}
		return
	}

	c.typingIndicatorBar.SetVisible(true)
	c.startTypingDotsAnimation()
	if c.autoScrollBottom {
		c.scrollToBottomAnimated()
	}

	c.typingTimeoutSource = glib.TimeoutAdd(5000, func() bool {
		c.setTypingIndicator(false)
		return false
	})
}

func (c *MainController) startTypingDotsAnimation() {
	if c.typingAnimSource != 0 {
		return
	}
	dotIdx := 0
	c.typingAnimSource = glib.TimeoutAdd(280, func() bool {
		if c.typingIndicatorBar == nil || !c.typingIndicatorBar.Visible() {
			c.typingAnimSource = 0
			return false
		}
		dots := []*gtk.Label{c.typingDot1, c.typingDot2, c.typingDot3}
		for i, dot := range dots {
			if dot != nil {
				if i == dotIdx {
					dot.SetOpacity(1.0)
					dot.SetMarginBottom(4)
				} else {
					dot.SetOpacity(0.35)
					dot.SetMarginBottom(0)
				}
			}
		}
		dotIdx = (dotIdx + 1) % 3
		return true
	})
}
