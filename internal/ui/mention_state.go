package ui

import (
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

type mentionPopupState struct {
	popover         *gtk.Popover
	listBox         *gtk.ListBox
	scrolled        *gtk.ScrolledWindow
	activeMembers   []GroupMemberItem
	filteredMembers []GroupMemberItem
	selectedIndex   int
	atStartPos      int
	insertedMentions map[string]string // display name -> JID
}

var activeMentionState = &mentionPopupState{
	selectedIndex:    0,
	atStartPos:       -1,
	insertedMentions: make(map[string]string),
}

func (c *MainController) isMentionPopupVisible() bool {
	return activeMentionState.popover != nil && activeMentionState.popover.Visible() && len(activeMentionState.filteredMembers) > 0
}

func (c *MainController) hideMentionPopup() {
	if activeMentionState.popover != nil {
		activeMentionState.popover.Popdown()
	}
	activeMentionState.filteredMembers = nil
	activeMentionState.selectedIndex = 0
	activeMentionState.atStartPos = -1
}

func (c *MainController) navigateMentionPopup(delta int) {
	n := len(activeMentionState.filteredMembers)
	if n == 0 {
		return
	}
	activeMentionState.selectedIndex = (activeMentionState.selectedIndex + delta + n) % n
	c.renderMentionPopupSelection()
}

func (c *MainController) renderMentionPopupSelection() {
	if activeMentionState.listBox == nil {
		return
	}
	idx := 0
	for child := activeMentionState.listBox.FirstChild(); child != nil; child = gtk.BaseWidget(child).NextSibling() {
		if row, ok := child.(*gtk.ListBoxRow); ok {
			if idx == activeMentionState.selectedIndex {
				row.AddCSSClass("selected")
				activeMentionState.listBox.SelectRow(row)
			} else {
				row.RemoveCSSClass("selected")
			}
		}
		idx++
	}
}
