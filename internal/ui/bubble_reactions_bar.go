package ui

import (
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/store"
)

// createReactionsBar creates the horizontal quick reactions emoji bar for the context menu.
func (c *MainController) createReactionsBar(msg store.StoredMessage, parentPopover *gtk.Popover, anchorWidget gtk.Widgetter) *gtk.Box {
	bar := gtk.NewBox(gtk.OrientationHorizontal, 4)
	bar.AddCSSClass("context-reactions-bar")
	bar.SetHAlign(gtk.AlignCenter)

	commonEmojis := []string{"👍", "❤️", "😂", "😮", "😢", "🙏"}

	ourJID := ""
	if c.client != nil && c.client.Store != nil && c.client.Store.ID != nil {
		ourJID = c.client.Store.ID.ToNonAD().String()
	}

	var myReaction string
	for _, r := range msg.Reactions {
		if r.FromMe || (ourJID != "" && r.SenderJID == ourJID) || c.isSelfJID(r.SenderJID) {
			myReaction = r.Emoji
			break
		}
	}

	mCopy := msg
	for _, emoji := range commonEmojis {
		e := emoji
		btn := gtk.NewButton()
		addCSSClasses(btn, "flat", "circular", "reaction-quick-btn")
		lbl := gtk.NewLabel(e)
		lbl.SetHAlign(gtk.AlignCenter)
		lbl.SetVAlign(gtk.AlignCenter)
		btn.SetChild(lbl)
		if e == myReaction {
			btn.AddCSSClass("active")
		}

		btn.ConnectClicked(func() {
			parentPopover.Popdown()
			c.sendReaction(mCopy, e)
		})
		bar.Append(btn)
	}

	moreBtn := gtk.NewButton()
	addCSSClasses(moreBtn, "flat", "circular", "reaction-quick-btn")
	moreBtnIcon := gtk.NewImageFromIconName("list-add-symbolic")
	moreBtnIcon.SetHAlign(gtk.AlignCenter)
	moreBtnIcon.SetVAlign(gtk.AlignCenter)
	moreBtn.SetChild(moreBtnIcon)
	moreBtn.SetTooltipText("More emojis")

	moreBtn.ConnectClicked(func() {
		parentPopover.Popdown()
		w := gtk.BaseWidget(anchorWidget)
		chooser := gtk.NewEmojiChooser()
		chooser.SetParent(w)
		chooser.SetHasArrow(true)
		chooser.ConnectEmojiPicked(func(text string) {
			chooser.Popdown()
			c.sendReaction(mCopy, text)
		})
		chooser.Popup()
	})
	bar.Append(moreBtn)

	return bar
}
