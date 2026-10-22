package ui

import (
	"fmt"
	"strings"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"go.mau.fi/whatsmeow/types"
)

func (c *MainController) showAddGroupMembersDialog(groupJID string, existing []types.GroupParticipant, onDone func()) {
	if c.client == nil && (c.store == nil || !c.store.IsDemo()) {
		return
	}
	inGroup := make(map[string]bool)
	for _, p := range existing {
		inGroup[p.JID.ToNonAD().String()] = true
		if !p.PhoneNumber.IsEmpty() {
			inGroup[p.PhoneNumber.ToNonAD().String()] = true
		}
	}

	var candidates []groupContactItem
	for _, cnt := range c.collectSelectableContacts() {
		if !inGroup[cnt.JID] {
			candidates = append(candidates, cnt)
		}
	}

	win := gtk.NewWindow()
	win.SetTransientFor(&c.window.Window)
	win.SetModal(true)
	win.SetDefaultSize(420, 520)
	win.SetTitle("Add Participants")

	hb := adw.NewHeaderBar()
	cancelBtn := gtk.NewButtonWithLabel("Cancel")
	cancelBtn.AddCSSClass("flat")
	cancelBtn.ConnectClicked(func() { win.Close() })
	hb.PackStart(cancelBtn)

	addBtn := gtk.NewButtonWithLabel("Add")
	addBtn.AddCSSClass("suggested-action")
	hb.PackEnd(addBtn)
	win.SetTitlebar(hb)

	scrolled := gtk.NewScrolledWindow()
	scrolled.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)
	box := gtk.NewBox(gtk.OrientationVertical, 12)
	box.SetMarginTop(14)
	box.SetMarginBottom(16)
	box.SetMarginStart(14)
	box.SetMarginEnd(14)

	search := gtk.NewSearchEntry()
	search.SetPlaceholderText("Search contacts…")
	grp := adw.NewPreferencesGroup()
	grp.SetTitle("Select Contacts (0 selected)")
	box.Append(search)
	box.Append(grp)

	for i := range candidates {
		row := c.buildContactRow(&candidates[i])
		candidates[i].Check.ConnectToggled(func() {
			n := 0
			for j := range candidates {
				if candidates[j].Check != nil && candidates[j].Check.Active() {
					n++
				}
			}
			grp.SetTitle(fmt.Sprintf("Select Contacts (%d selected)", n))
		})
		grp.Add(row)
	}

	search.ConnectSearchChanged(func() {
		q := strings.TrimSpace(search.Text())
		for i := range candidates {
			m1, _ := FuzzyMatch(q, candidates[i].DisplayName)
			m2, _ := FuzzyMatch(q, candidates[i].Subtitle)
			candidates[i].Row.SetVisible(q == "" || m1 || m2)
		}
	})

	c.bindAddMembersSubmit(win, addBtn, groupJID, candidates, onDone)
	scrolled.SetChild(box)
	win.SetChild(scrolled)
	win.Present()
}
