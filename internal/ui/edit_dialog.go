package ui

import (
	"context"
	"fmt"
	"log"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/store"
)

// showEditHistoryDialog displays a modal dialog showing previous versions of a message.
func (c *MainController) showEditHistoryDialog(msg store.StoredMessage) {
	var edits []store.MessageEdit
	if c.store != nil {
		var err error
		edits, err = c.store.GetMessageEdits(context.Background(), msg.ChatJID, msg.ID)
		if err != nil {
			log.Printf("fetching message edits: %v", err)
		}
	}

	win := adw.NewWindow()
	win.SetTransientFor(&c.window.Window)
	win.SetModal(true)
	win.SetTitle("Edit History")
	win.SetDefaultSize(440, 480)

	tbView := adw.NewToolbarView()
	hb := adw.NewHeaderBar()
	hb.SetTitleWidget(adw.NewWindowTitle("Edit History", cleanPhoneNumber(msg.ChatJID)))
	tbView.AddTopBar(hb)

	scrolled := gtk.NewScrolledWindow()
	scrolled.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)
	scrolled.SetVExpand(true)

	mainBox := gtk.NewBox(gtk.OrientationVertical, 12)
	mainBox.SetMarginTop(16)
	mainBox.SetMarginBottom(16)
	mainBox.SetMarginStart(16)
	mainBox.SetMarginEnd(16)

	for i, e := range edits {
		versionLabel := fmt.Sprintf("Edit %d", i+1)
		if i == 0 {
			versionLabel = "Original Message"
		}
		appendEditHistoryCard(mainBox, versionLabel, e.EditedAt.Format("02 Jan 15:04"), e.Text, false)
	}

	appendEditHistoryCard(mainBox, "Latest Version (Current)", msg.Timestamp.Format("02 Jan 15:04"), msg.Text, true)

	scrolled.SetChild(mainBox)
	tbView.SetContent(scrolled)
	win.SetContent(tbView)
	win.Present()
}
