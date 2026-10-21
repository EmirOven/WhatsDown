package ui

import (
	"fmt"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

func buildPollQuestionGroup() (*adw.PreferencesGroup, *gtk.Entry) {
	group := adw.NewPreferencesGroup()
	group.SetTitle("Question")
	row := adw.NewActionRow()
	row.SetTitle("Question")
	entry := gtk.NewEntry()
	entry.SetPlaceholderText("Ask a question…")
	entry.SetHExpand(true)
	row.AddSuffix(entry)
	row.SetActivatableWidget(entry)
	group.Add(row)
	return group, entry
}

func buildPollSettingsGroup() (*adw.PreferencesGroup, *gtk.Switch) {
	group := adw.NewPreferencesGroup()
	group.SetTitle("Settings")
	row := adw.NewActionRow()
	row.SetTitle("Allow multiple answers")
	row.SetSubtitle("Voters can pick more than one option")
	sw := gtk.NewSwitch()
	sw.SetVAlign(gtk.AlignCenter)
	row.AddSuffix(sw)
	row.SetActivatableWidget(sw)
	group.Add(row)
	return group, sw
}

func buildPollOptionsGroup(optionEntries *[]*gtk.Entry) (*adw.PreferencesGroup, *gtk.Button) {
	optionsGroup := adw.NewPreferencesGroup()
	optionsGroup.SetTitle("Options")
	addOptionRow := func(placeholder string) {
		row := adw.NewActionRow()
		row.SetTitle(placeholder)
		entry := gtk.NewEntry()
		entry.SetPlaceholderText("Add option…")
		entry.SetHExpand(true)
		row.AddSuffix(entry)
		row.SetActivatableWidget(entry)
		optionsGroup.Add(row)
		*optionEntries = append(*optionEntries, entry)
	}
	addOptionRow("Option 1")
	addOptionRow("Option 2")

	addBtn := gtk.NewButtonWithLabel("+ Add Option")
	addCSSClasses(addBtn, "flat", "pill")
	addBtn.SetHAlign(gtk.AlignStart)
	addBtn.ConnectClicked(func() {
		if len(*optionEntries) < 12 {
			addOptionRow(fmt.Sprintf("Option %d", len(*optionEntries)+1))
			if len(*optionEntries) >= 12 {
				addBtn.SetSensitive(false)
			}
		}
	})
	return optionsGroup, addBtn
}
