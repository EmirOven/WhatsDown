package ui

import (
	"strings"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// showCreatePollDialog opens a modal dialog allowing the user to create and send a poll.
func (c *MainController) showCreatePollDialog() {
	if c.currentChat == nil {
		return
	}
	dialog := gtk.NewWindow()
	dialog.SetTransientFor(&c.window.Window)
	dialog.SetModal(true)
	dialog.SetDefaultSize(440, 520)
	dialog.SetTitle("Create Poll")

	headerBar := adw.NewHeaderBar()
	cancelBtn := gtk.NewButtonWithLabel("Cancel")
	cancelBtn.AddCSSClass("flat")
	cancelBtn.ConnectClicked(func() { dialog.Close() })
	headerBar.PackStart(cancelBtn)

	createBtn := gtk.NewButtonWithLabel("Create")
	createBtn.AddCSSClass("suggested-action")
	headerBar.PackEnd(createBtn)
	dialog.SetTitlebar(headerBar)

	scrolledWindow := gtk.NewScrolledWindow()
	scrolledWindow.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)

	mainBox := gtk.NewBox(gtk.OrientationVertical, 16)
	mainBox.SetMarginTop(16)
	mainBox.SetMarginBottom(20)
	mainBox.SetMarginStart(16)
	mainBox.SetMarginEnd(16)

	questionGroup, questionEntry := buildPollQuestionGroup()
	mainBox.Append(questionGroup)

	var optionEntries []*gtk.Entry
	optionsGroup, addBtn := buildPollOptionsGroup(&optionEntries)
	mainBox.Append(optionsGroup)
	mainBox.Append(addBtn)

	settingsGroup, multipleSwitch := buildPollSettingsGroup()
	mainBox.Append(settingsGroup)

	errorLabel := gtk.NewLabel("")
	addCSSClasses(errorLabel, "caption", "error-label")
	errorLabel.SetHAlign(gtk.AlignCenter)
	errorLabel.SetVisible(false)
	mainBox.Append(errorLabel)

	scrolledWindow.SetChild(mainBox)
	dialog.SetChild(scrolledWindow)

	createBtn.ConnectClicked(func() {
		q := strings.TrimSpace(questionEntry.Text())
		if q == "" {
			errorLabel.SetText("Please enter a question.")
			errorLabel.SetVisible(true)
			return
		}
		var opts []string
		for _, e := range optionEntries {
			if txt := strings.TrimSpace(e.Text()); txt != "" {
				opts = append(opts, txt)
			}
		}
		if len(opts) < 2 {
			errorLabel.SetText("Please enter at least 2 options.")
			errorLabel.SetVisible(true)
			return
		}
		dialog.Close()
		c.sendPoll(q, opts, multipleSwitch.Active())
	})
	dialog.Present()
}
