package ui

import (
	"fmt"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotk4/pkg/pango"
	"postbetasoftware.com/whatsdown/internal/avatar"
	"postbetasoftware.com/whatsdown/internal/store"
)

func (c *MainController) createCallRow(call store.CallRecord) *gtk.ListBoxRow {
	row := gtk.NewListBoxRow()
	row.AddCSSClass("chat-row")
	row.AddCSSClass("call-row")

	mainBox := gtk.NewBox(gtk.OrientationHorizontal, 12)
	mainBox.SetHExpand(true)

	displayName := call.CallerName
	if displayName == "" {
		displayName = cleanPhoneNumber(call.CallerJID)
	}

	av := adw.NewAvatar(40, displayName, true)
	av.SetVAlign(gtk.AlignCenter)
	avatar.LoadAvatar(c.client, call.CallerJID, displayName, av)
	mainBox.Append(av)

	textBox := gtk.NewBox(gtk.OrientationVertical, 4)
	textBox.SetHExpand(true)
	textBox.SetVAlign(gtk.AlignCenter)

	titleLabel := gtk.NewLabel(displayName)
	titleLabel.SetXAlign(0)
	titleLabel.AddCSSClass("chat-row-title")
	titleLabel.SetEllipsize(pango.EllipsizeEnd)
	if call.Outcome == store.CallOutcomeMissed && !call.FromMe {
		titleLabel.AddCSSClass("call-title-missed")
	}
	textBox.Append(titleLabel)

	subBox := gtk.NewBox(gtk.OrientationHorizontal, 6)
	subBox.SetVAlign(gtk.AlignCenter)

	iconName := "call-incoming-symbolic"
	iconClass := "call-icon-incoming"
	if call.FromMe {
		iconName = "call-outgoing-symbolic"
		iconClass = "call-icon-outgoing"
	} else if call.Outcome == store.CallOutcomeMissed {
		iconName = "call-missed-symbolic"
		iconClass = "call-icon-missed"
	}

	callIcon := gtk.NewImageFromIconName(iconName)
	callIcon.AddCSSClass(iconClass)
	subBox.Append(callIcon)

	if call.IsVideo {
		vidIcon := gtk.NewImageFromIconName("camera-video-symbolic")
		vidIcon.AddCSSClass("dim-label")
		subBox.Append(vidIcon)
	}

	detailText := formatChatTime(call.Timestamp)
	if call.DurationSecs > 0 {
		detailText = fmt.Sprintf("%s (%s)", detailText, store.FormatCallDuration(call.DurationSecs))
	} else if call.Outcome == store.CallOutcomeMissed {
		detailText = fmt.Sprintf("%s · Missed", detailText)
	}
	subLabel := gtk.NewLabel(detailText)
	subLabel.SetXAlign(0)
	subLabel.AddCSSClass("chat-row-subtitle")
	if call.Outcome == store.CallOutcomeMissed && !call.FromMe {
		subLabel.AddCSSClass("call-subtitle-missed")
	}
	subBox.Append(subLabel)
	textBox.Append(subBox)
	mainBox.Append(textBox)

	actionBtn := gtk.NewButton()
	actionBtn.AddCSSClass("flat")
	actionBtn.AddCSSClass("circular")
	actionBtn.SetVAlign(gtk.AlignCenter)
	actionBtn.SetChild(gtk.NewImageFromIconName("chat-message-new-symbolic"))
	actionBtn.SetTooltipText("Open chat")
	actionBtn.ConnectClicked(func() {
		c.openCallChat(call)
	})
	mainBox.Append(actionBtn)

	row.SetChild(mainBox)
	c.attachCallContextMenu(row, call)
	return row
}
