package ui

import (
	"fmt"
	"strings"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/store"
)

// buildReactionPill builds the small reaction pill that sits at the bottom corner of a message bubble.
func (c *MainController) buildReactionPill(msg store.StoredMessage) *gtk.Button {
	if len(msg.Reactions) == 0 {
		return nil
	}

	btn := gtk.NewButton()
	addCSSClasses(btn, "flat", "reaction-pill")
	btn.SetHAlign(gtk.AlignCenter)
	btn.SetVAlign(gtk.AlignCenter)

	hasMine := false
	emojiCounts := make(map[string]int)
	var distinctEmojis []string
	for _, r := range msg.Reactions {
		if r.FromMe || c.isSelfJID(r.SenderJID) {
			hasMine = true
		}
		if emojiCounts[r.Emoji] == 0 {
			distinctEmojis = append(distinctEmojis, r.Emoji)
		}
		emojiCounts[r.Emoji]++
	}

	if hasMine {
		btn.AddCSSClass("reaction-pill-mine")
	}

	pillBox := gtk.NewBox(gtk.OrientationHorizontal, 3)
	pillBox.SetHAlign(gtk.AlignCenter)
	pillBox.SetVAlign(gtk.AlignCenter)

	emojiDisplay := ""
	for i, e := range distinctEmojis {
		if i >= 3 {
			break
		}
		emojiDisplay += e
	}
	emojiLabel := gtk.NewLabel(emojiDisplay)
	emojiLabel.AddCSSClass("reaction-pill-emojis")
	emojiLabel.SetHAlign(gtk.AlignCenter)
	emojiLabel.SetVAlign(gtk.AlignCenter)
	pillBox.Append(emojiLabel)

	if totalCount := len(msg.Reactions); totalCount > 1 {
		countLabel := gtk.NewLabel(fmt.Sprintf("%d", totalCount))
		countLabel.AddCSSClass("reaction-pill-count")
		countLabel.SetHAlign(gtk.AlignCenter)
		countLabel.SetVAlign(gtk.AlignCenter)
		pillBox.Append(countLabel)
	}

	btn.SetChild(pillBox)

	var tooltipNames []string
	for _, r := range msg.Reactions {
		name := "You"
		if !r.FromMe && !c.isSelfJID(r.SenderJID) {
			name = c.resolveContactName(r.SenderJID)
			if name == "" {
				name = r.SenderName
			}
		}
		tooltipNames = append(tooltipNames, fmt.Sprintf("%s (%s)", name, r.Emoji))
	}
	btn.SetTooltipText(strings.Join(tooltipNames, ", "))

	mCopy := msg
	btn.ConnectClicked(func() {
		c.showReactionsListPopover(btn, mCopy)
	})

	return btn
}
