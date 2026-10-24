package ui

import (
	"fmt"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/store"
)

type messageSelectionState struct {
	active          bool
	chat            string
	selected        map[string]bool
	checks          map[string]*gtk.CheckButton
	toolbar         *gtk.Box
	count           *gtk.Label
	forward, remove *gtk.Button
}

func (c *MainController) ensureMessageSelection() *messageSelectionState {
	if c.messageSelection == nil {
		c.messageSelection = &messageSelectionState{selected: make(map[string]bool), checks: make(map[string]*gtk.CheckButton)}
	}
	return c.messageSelection
}

func (c *MainController) showMessageSelection(initial string) {
	if c.currentChat == nil {
		return
	}
	state := c.ensureMessageSelection()
	state.active = true
	state.chat = c.currentChat.JID
	state.selected[initial] = true
	if state.toolbar == nil && c.chatPage != nil {
		bar := gtk.NewBox(gtk.OrientationHorizontal, 8)
		bar.AddCSSClass("toolbar")
		bar.SetFocusable(true)
		bar.SetMarginStart(12)
		bar.SetMarginEnd(12)
		close := gtk.NewButtonFromIconName("window-close-symbolic")
		close.SetTooltipText("Cancel message selection (Esc)")
		close.AddCSSClass("flat")
		c.trackWidgetSignal(bar, close, close.ConnectClicked(c.cancelMessageSelection))
		bar.Append(close)
		state.count = gtk.NewLabel("")
		state.count.SetHExpand(true)
		state.count.SetXAlign(0)
		state.count.AddCSSClass("heading")
		bar.Append(state.count)
		state.forward = gtk.NewButtonWithLabel("Forward…")
		c.trackWidgetSignal(bar, state.forward, state.forward.ConnectClicked(func() {
			messages := c.selectedMessages()
			if len(messages) == 0 {
				return
			}
			c.showForwardMessages(messages)
		}))
		bar.Append(state.forward)
		state.remove = gtk.NewButtonWithLabel("Delete…")
		c.trackWidgetSignal(bar, state.remove, state.remove.ConnectClicked(func() { c.confirmDeleteMessages(c.selectedMessages()) }))
		bar.Append(state.remove)
		state.toolbar = bar
		c.chatPage.Prepend(bar)
	}
	c.updateMessageSelectionUI()
	if state.toolbar != nil {
		state.toolbar.GrabFocus()
	}
}

func (c *MainController) cancelMessageSelection() {
	if c.messageSelection == nil {
		return
	}
	c.messageSelection.active = false
	c.messageSelection.selected = make(map[string]bool)
	c.updateMessageSelectionUI()
}

func (c *MainController) selectedMessages() []store.StoredMessage {
	if c.messageSelection == nil {
		return nil
	}
	var messages []store.StoredMessage
	for _, msg := range c.currentMessages {
		if c.messageSelection.selected[msg.ID] {
			messages = append(messages, msg)
		}
	}
	return messages
}

func (c *MainController) updateMessageSelectionUI() {
	state := c.messageSelection
	if state == nil {
		return
	}
	for id, check := range state.checks {
		check.SetVisible(state.active)
		selected := state.active && state.selected[id]
		// Avoid recursive toggled callbacks when synchronizing multiple rows.
		if check.Active() != selected {
			check.SetActive(selected)
		}
		if row := c.messageRowMap[id]; row != nil {
			if selected {
				row.AddCSSClass("message-selected")
			} else {
				row.RemoveCSSClass("message-selected")
			}
		}
	}
	if c.chatInputStack != nil {
		c.chatInputStack.SetVisible(!state.active)
	}
	if state.toolbar == nil {
		return
	}
	state.toolbar.SetVisible(state.active)
	messages := c.selectedMessages()
	state.count.SetText(fmt.Sprintf("%d selected", len(messages)))
	canForward := len(messages) > 0
	for _, msg := range messages {
		if _, err := buildForwardMessage(msg); err != nil {
			canForward = false
			break
		}
	}
	state.forward.SetSensitive(canForward)
	state.remove.SetSensitive(len(messages) > 0)
}

func (c *MainController) pruneMessageSelection() {
	if c.messageSelection == nil {
		return
	}
	present := make(map[string]bool, len(c.currentMessages))
	for _, msg := range c.currentMessages {
		present[msg.ID] = true
	}
	for id := range c.messageSelection.selected {
		if !present[id] {
			delete(c.messageSelection.selected, id)
		}
	}
	c.updateMessageSelectionUI()
}

func (c *MainController) wrapMessageSelectionRow(row *gtk.ListBoxRow, content *gtk.Box, msg store.StoredMessage) *gtk.Box {
	state := c.ensureMessageSelection()
	box := gtk.NewBox(gtk.OrientationHorizontal, 8)
	box.SetHExpand(true)
	check := gtk.NewCheckButton()
	check.SetTooltipText("Select message")
	check.SetVAlign(gtk.AlignCenter)
	check.SetMarginStart(12)
	check.SetVisible(state.active)
	check.SetActive(state.active && state.selected[msg.ID])
	if state.active && state.selected[msg.ID] {
		row.AddCSSClass("message-selected")
	}
	state.checks[msg.ID] = check
	c.trackWidgetSignal(row, check, check.ConnectToggled(func() {
		if !state.active {
			return
		}
		if check.Active() {
			state.selected[msg.ID] = true
		} else {
			delete(state.selected, msg.ID)
		}
		c.updateMessageSelectionUI()
	}))
	box.Append(check)
	content.SetHExpand(true)
	box.Append(content)
	click := gtk.NewGestureClick()
	click.SetButton(gdk.BUTTON_PRIMARY)
	click.SetPropagationPhase(gtk.PhaseCapture)
	c.trackWidgetSignal(row, click, click.ConnectPressed(func(_ int, _, _ float64) {
		if !state.active {
			return
		}
		click.SetState(gtk.EventSequenceClaimed)
		check.SetActive(!check.Active())
	}))
	row.AddController(click)
	return box
}
