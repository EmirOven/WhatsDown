package ui

import (
	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

func (c *MainController) showAboutDialog() *adw.AboutDialog {
	about := adw.NewAboutDialog()
	about.SetApplicationName("WhatsDown")
	about.SetApplicationIcon("com.postbetasoftware.WhatsDown")
	about.SetDeveloperName("Emir Öven")
	about.SetDevelopers([]string{"Emir Öven https://github.com/EmirOven"})
	about.SetVersion("3")
	about.SetComments("Native Linux WhatsApp client built with GTK 4, Libadwaita and Go.")
	about.SetLicenseType(gtk.LicenseMITX11)
	about.SetWebsite("https://github.com/EmirOven/WhatsDown")
	about.SetIssueURL("https://github.com/EmirOven/WhatsDown/issues")
	about.Present(c.window)
	return about
}
