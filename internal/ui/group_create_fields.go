package ui

import (
	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

func (c *MainController) buildGroupAvatarPicker(parent *gtk.Window, selectedPath *string) *gtk.Box {
	box := gtk.NewBox(gtk.OrientationVertical, 6)
	box.SetHAlign(gtk.AlignCenter)

	av := adw.NewAvatar(72, "Group", true)
	av.SetIconName("system-users-symbolic")
	av.SetShowInitials(false)

	btn := gtk.NewButton()
	addCSSClasses(btn, "flat", "circular")
	btn.SetChild(av)
	btn.SetTooltipText("Choose Group Avatar")
	btn.SetHAlign(gtk.AlignCenter)

	lbl := gtk.NewLabel("Click to set avatar")
	addCSSClasses(lbl, "caption", "dim-label")

	btn.ConnectClicked(func() {
		chooser := gtk.NewFileChooserNative("Choose Group Avatar", parent, gtk.FileChooserActionOpen, "_Select", "_Cancel")
		filter := gtk.NewFileFilter()
		filter.SetName("Images")
		filter.AddMIMEType("image/*")
		filter.AddPattern("*.jpg")
		filter.AddPattern("*.jpeg")
		filter.AddPattern("*.png")
		filter.AddPattern("*.webp")
		chooser.SetFilter(filter)

		chooser.ConnectResponse(func(resp int) {
			if resp == int(gtk.ResponseAccept) {
				if file := chooser.File(); file != nil {
					if path := file.Path(); path != "" {
						*selectedPath = path
						glib.IdleAdd(func() {
							if tex, err := gdk.NewTextureFromFilename(path); err == nil {
								av.SetCustomImage(tex)
							}
						})
					}
				}
			}
		})
		chooser.Show()
	})

	box.Append(btn)
	box.Append(lbl)
	return box
}

func buildGroupInfoGroup() (*adw.PreferencesGroup, *gtk.Entry, *gtk.Entry) {
	group := adw.NewPreferencesGroup()
	group.SetTitle("Group Details")

	nameRow := adw.NewActionRow()
	nameRow.SetTitle("Name")
	nameEntry := gtk.NewEntry()
	nameEntry.SetPlaceholderText("Group name (max 25 chars)…")
	nameEntry.SetMaxLength(25)
	nameEntry.SetHExpand(true)
	nameEntry.SetVAlign(gtk.AlignCenter)
	nameRow.AddSuffix(nameEntry)
	nameRow.SetActivatableWidget(nameEntry)
	group.Add(nameRow)

	descRow := adw.NewActionRow()
	descRow.SetTitle("Description")
	descEntry := gtk.NewEntry()
	descEntry.SetPlaceholderText("Optional group description…")
	descEntry.SetHExpand(true)
	descEntry.SetVAlign(gtk.AlignCenter)
	descRow.AddSuffix(descEntry)
	descRow.SetActivatableWidget(descEntry)
	group.Add(descRow)

	return group, nameEntry, descEntry
}
