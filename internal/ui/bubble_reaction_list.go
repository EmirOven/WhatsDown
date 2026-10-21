package ui

import (
	"fmt"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/store"
)

// showReactionsListPopover displays a popover with the list of who gave which reaction.
func (c *MainController) showReactionsListPopover(parent gtk.Widgetter, msg store.StoredMessage) {
	w := gtk.BaseWidget(parent)
	popover := gtk.NewPopover()
	popover.SetParent(w)
	popover.SetHasArrow(true)
	popover.AddCSSClass("reactions-popover")

	mainBox := gtk.NewBox(gtk.OrientationVertical, 6)
	mainBox.SetMarginTop(8)
	mainBox.SetMarginBottom(8)
	mainBox.SetMarginStart(10)
	mainBox.SetMarginEnd(10)

	headerBox := gtk.NewBox(gtk.OrientationHorizontal, 8)
	titleLabel := gtk.NewLabel("Reactions")
	titleLabel.AddCSSClass("heading")
	titleLabel.SetHAlign(gtk.AlignStart)
	titleLabel.SetHExpand(true)
	headerBox.Append(titleLabel)
	mainBox.Append(headerBox)

	emojiCounts := make(map[string]int)
	var distinctEmojis []string
	for _, r := range msg.Reactions {
		if emojiCounts[r.Emoji] == 0 {
			distinctEmojis = append(distinctEmojis, r.Emoji)
		}
		emojiCounts[r.Emoji]++
	}

	listBox := gtk.NewListBox()
	listBox.SetSelectionMode(gtk.SelectionNone)
	addCSSClasses(listBox, "boxed-list", "reactions-list")

	if len(distinctEmojis) > 1 {
		chipsBox := gtk.NewBox(gtk.OrientationHorizontal, 4)
		chipsBox.SetMarginBottom(4)

		allBtn := gtk.NewButtonWithLabel(fmt.Sprintf("All %d", len(msg.Reactions)))
		addCSSClasses(allBtn, "flat", "circular", "active")
		chipsBox.Append(allBtn)

		var chipBtns []*gtk.Button
		chipBtns = append(chipBtns, allBtn)

		allBtn.ConnectClicked(func() {
			for _, b := range chipBtns {
				b.RemoveCSSClass("active")
			}
			allBtn.AddCSSClass("active")
			c.populateReactionsList(listBox, popover, msg, "")
		})

		for _, e := range distinctEmojis {
			eCopy := e
			count := emojiCounts[eCopy]
			chipBtn := gtk.NewButtonWithLabel(fmt.Sprintf("%s %d", eCopy, count))
			addCSSClasses(chipBtn, "flat", "circular")
			chipsBox.Append(chipBtn)
			chipBtns = append(chipBtns, chipBtn)

			chipBtn.ConnectClicked(func() {
				for _, b := range chipBtns {
					b.RemoveCSSClass("active")
				}
				chipBtn.AddCSSClass("active")
				c.populateReactionsList(listBox, popover, msg, eCopy)
			})
		}
		mainBox.Append(chipsBox)
	}

	c.populateReactionsList(listBox, popover, msg, "")

	scrolled := gtk.NewScrolledWindow()
	scrolled.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)
	scrolled.SetPropagateNaturalHeight(true)
	scrolled.SetMaxContentHeight(260)
	scrolled.SetChild(listBox)

	mainBox.Append(scrolled)
	popover.SetChild(mainBox)
	popover.Popup()
}
