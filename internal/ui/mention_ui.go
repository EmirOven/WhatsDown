package ui

import (
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

func (c *MainController) showMentionPopup(members []GroupMemberItem) {
	if c.messageEntry == nil {
		return
	}

	if activeMentionState.popover == nil {
		c.createMentionPopover()
	}

	for child := activeMentionState.listBox.FirstChild(); child != nil; {
		next := gtk.BaseWidget(child).NextSibling()
		activeMentionState.listBox.Remove(child)
		child = next
	}

	for i, item := range members {
		row := c.createMentionRow(item, i)
		activeMentionState.listBox.Append(row)
	}

	if activeMentionState.selectedIndex >= len(members) {
		activeMentionState.selectedIndex = 0
	}
	c.renderMentionPopupSelection()

	if !activeMentionState.popover.Visible() {
		activeMentionState.popover.Popup()
	}
}

func (c *MainController) createMentionPopover() {
	pop := gtk.NewPopover()
	pop.SetParent(c.messageEntry)
	pop.SetPosition(gtk.PosTop)
	pop.SetHasArrow(false)
	pop.SetAutohide(false)
	pop.AddCSSClass("mention-popover")

	listBox := gtk.NewListBox()
	listBox.SetSelectionMode(gtk.SelectionSingle)
	listBox.AddCSSClass("mention-list")

	listBox.ConnectRowActivated(func(row *gtk.ListBoxRow) {
		idx := row.Index()
		if idx >= 0 && idx < len(activeMentionState.filteredMembers) {
			activeMentionState.selectedIndex = idx
			c.commitSelectedMention()
		}
	})

	scrolled := gtk.NewScrolledWindow()
	scrolled.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)
	scrolled.SetPropagateNaturalHeight(true)
	scrolled.SetMaxContentHeight(240) // 5 entries @ ~48px each
	scrolled.SetSizeRequest(260, -1)
	scrolled.SetChild(listBox)

	pop.SetChild(scrolled)
	activeMentionState.popover = pop
	activeMentionState.listBox = listBox
	activeMentionState.scrolled = scrolled
}
