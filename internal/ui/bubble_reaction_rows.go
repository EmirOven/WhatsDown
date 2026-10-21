package ui

import (
	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/store"
)

func (c *MainController) populateReactionsList(listBox *gtk.ListBox, popover *gtk.Popover, msg store.StoredMessage, filterEmoji string) {
	for {
		row := listBox.RowAtIndex(0)
		if row == nil {
			break
		}
		listBox.Remove(row)
	}

	for _, r := range msg.Reactions {
		if filterEmoji != "" && r.Emoji != filterEmoji {
			continue
		}

		rCopy := r
		row := gtk.NewListBoxRow()
		row.SetSelectable(false)
		row.AddCSSClass("reactions-list-row")

		rowBox := gtk.NewBox(gtk.OrientationHorizontal, 10)
		rowBox.SetMarginTop(6)
		rowBox.SetMarginBottom(6)
		rowBox.SetMarginStart(8)
		rowBox.SetMarginEnd(8)
		rowBox.SetVAlign(gtk.AlignCenter)

		isMine := rCopy.FromMe || c.isSelfJID(rCopy.SenderJID)
		name := "You"
		if !isMine {
			name = c.resolveContactName(rCopy.SenderJID)
			if name == "" {
				name = rCopy.SenderName
			}
		}

		avatar := adw.NewAvatar(32, name, true)
		rowBox.Append(avatar)

		nameBox := gtk.NewBox(gtk.OrientationVertical, 2)
		nameBox.SetHExpand(true)
		nameLabel := gtk.NewLabel(name)
		nameLabel.SetXAlign(0)
		nameLabel.AddCSSClass("bold")
		nameBox.Append(nameLabel)

		if isMine {
			subLabel := gtk.NewLabel("Click to remove")
			subLabel.SetXAlign(0)
			addCSSClasses(subLabel, "dim-label", "caption")
			nameBox.Append(subLabel)
		}
		rowBox.Append(nameBox)

		emojiLbl := gtk.NewLabel(rCopy.Emoji)
		emojiLbl.AddCSSClass("reaction-list-emoji")
		emojiLbl.SetVAlign(gtk.AlignCenter)
		rowBox.Append(emojiLbl)

		row.SetChild(rowBox)

		if isMine {
			clickGesture := gtk.NewGestureClick()
			clickGesture.ConnectPressed(func(nPress int, x, y float64) {
				popover.Popdown()
				c.sendReaction(msg, "")
			})
			row.AddController(clickGesture)
		}

		listBox.Append(row)
	}
}
