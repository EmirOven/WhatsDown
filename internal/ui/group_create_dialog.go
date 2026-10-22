package ui

import (
	"fmt"
	"strings"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

func (c *MainController) showCreateGroupDialog() {
	if c.client == nil {
		return
	}

	dialog := gtk.NewWindow()
	dialog.SetTransientFor(&c.window.Window)
	dialog.SetModal(true)
	dialog.SetDefaultSize(460, 620)
	dialog.SetTitle("New Group")

	headerBar := adw.NewHeaderBar()
	cancelBtn := gtk.NewButtonWithLabel("Cancel")
	cancelBtn.AddCSSClass("flat")
	cancelBtn.ConnectClicked(func() { dialog.Close() })
	headerBar.PackStart(cancelBtn)

	createBtn := gtk.NewButtonWithLabel("Create")
	createBtn.AddCSSClass("suggested-action")
	headerBar.PackEnd(createBtn)
	dialog.SetTitlebar(headerBar)

	scrolled := gtk.NewScrolledWindow()
	scrolled.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)

	mainBox := gtk.NewBox(gtk.OrientationVertical, 16)
	mainBox.SetMarginTop(16)
	mainBox.SetMarginBottom(20)
	mainBox.SetMarginStart(16)
	mainBox.SetMarginEnd(16)

	var avatarPath string
	avatarBox := c.buildGroupAvatarPicker(dialog, &avatarPath)
	mainBox.Append(avatarBox)

	infoGroup, nameEntry, descEntry := buildGroupInfoGroup()
	mainBox.Append(infoGroup)

	membersGroup := adw.NewPreferencesGroup()
	membersGroup.SetTitle("Select Members (0 selected)")

	searchEntry := gtk.NewSearchEntry()
	searchEntry.SetPlaceholderText("Search contacts…")
	searchEntry.SetMarginBottom(8)
	mainBox.Append(searchEntry)
	mainBox.Append(membersGroup)

	contacts := c.collectSelectableContacts()
	updateCount := func() {
		cnt := 0
		for i := range contacts {
			if contacts[i].Check != nil && contacts[i].Check.Active() {
				cnt++
			}
		}
		membersGroup.SetTitle(fmt.Sprintf("Select Members (%d selected)", cnt))
	}

	for i := range contacts {
		row := c.buildContactRow(&contacts[i])
		contacts[i].Check.ConnectToggled(updateCount)
		membersGroup.Add(row)
	}

	searchEntry.ConnectSearchChanged(func() {
		q := strings.TrimSpace(searchEntry.Text())
		for i := range contacts {
			m1, _ := FuzzyMatch(q, contacts[i].DisplayName)
			m2, _ := FuzzyMatch(q, contacts[i].Subtitle)
			contacts[i].Row.SetVisible(q == "" || m1 || m2)
		}
	})

	errLbl := gtk.NewLabel("")
	addCSSClasses(errLbl, "caption", "error-label")
	errLbl.SetHAlign(gtk.AlignCenter)
	errLbl.SetVisible(false)
	mainBox.Append(errLbl)

	scrolled.SetChild(mainBox)
	dialog.SetChild(scrolled)

	c.bindCreateGroupAction(dialog, createBtn, nameEntry, descEntry, &avatarPath, contacts, errLbl)
	dialog.Present()
}
