package ui

import (
	"context"
	"fmt"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotk4/pkg/pango"
	"postbetasoftware.com/whatsdown/internal/store"
)

// showPollVotesDialog opens a dialog displaying who voted for each option in a poll.
func (c *MainController) showPollVotesDialog(msg store.StoredMessage, question string, options []string) {
	dialog := adw.NewDialog()
	dialog.SetTitle("Poll Details")
	dialog.SetContentWidth(420)
	dialog.SetContentHeight(500)

	toolbarView := adw.NewToolbarView()
	headerBar := adw.NewHeaderBar()
	toolbarView.AddTopBar(headerBar)

	scrolled := gtk.NewScrolledWindow()
	scrolled.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)
	scrolled.SetVExpand(true)

	contentBox := gtk.NewBox(gtk.OrientationVertical, 16)
	contentBox.SetMarginTop(18)
	contentBox.SetMarginBottom(24)
	contentBox.SetMarginStart(18)
	contentBox.SetMarginEnd(18)

	qLabel := gtk.NewLabel(question)
	qLabel.AddCSSClass("title-2")
	qLabel.SetWrap(true)
	qLabel.SetWrapMode(pango.WrapWordChar)
	qLabel.SetXAlign(0)
	contentBox.Append(qLabel)

	if c.store != nil {
		results, _ := c.store.GetPollResults(context.Background(), msg.ChatJID, msg.ID)
		ownJID := ""
		if c.client != nil && c.client.Store != nil && c.client.Store.ID != nil {
			ownJID = c.client.Store.ID.ToNonAD().String()
		}

		for _, opt := range options {
			group := adw.NewPreferencesGroup()
			cnt := 0
			var voters []string
			if results != nil {
				cnt = results.VoteCounts[opt]
				voters = results.VotersByOption[opt]
			}
			suffix := "votes"
			if cnt == 1 {
				suffix = "vote"
			}
			group.SetTitle(fmt.Sprintf("%s (%d %s)", opt, cnt, suffix))

			if len(voters) == 0 {
				row := adw.NewActionRow()
				row.SetTitle("No votes yet")
				row.AddCSSClass("dim-label")
				group.Add(row)
			} else {
				for _, v := range voters {
					row := adw.NewActionRow()
					name := "You"
					if v != "me" && (ownJID == "" || v != ownJID) {
						if resolved := c.resolveContactName(v); resolved != "" && resolved != "WhatsApp User" {
							name = resolved
						} else if clean := cleanPhoneNumber(v); clean != "" {
							name = clean
						} else {
							name = v
						}
					}
					row.SetTitle(name)
					row.AddPrefix(gtk.NewImageFromIconName("avatar-default-symbolic"))
					group.Add(row)
				}
			}
			contentBox.Append(group)
		}
	}

	scrolled.SetChild(contentBox)
	toolbarView.SetContent(scrolled)
	dialog.SetChild(toolbarView)
	dialog.Present(c.window)
}
