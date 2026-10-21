package ui

import "github.com/diamondburned/gotk4-adwaita/pkg/adw"

func (ctrl *MainController) bindThemeRow() {
	if ctrl.settingsThemeRow == nil {
		return
	}
	sm := adw.StyleManagerGetDefault()
	switch sm.ColorScheme() {
	case adw.ColorSchemeForceLight:
		ctrl.settingsThemeRow.SetSelected(1)
	case adw.ColorSchemeForceDark:
		ctrl.settingsThemeRow.SetSelected(2)
	default:
		ctrl.settingsThemeRow.SetSelected(0)
	}
	ctrl.settingsThemeRow.Connect("notify::selected", func() {
		switch ctrl.settingsThemeRow.Selected() {
		case 1:
			sm.SetColorScheme(adw.ColorSchemeForceLight)
		case 2:
			sm.SetColorScheme(adw.ColorSchemeForceDark)
		default:
			sm.SetColorScheme(adw.ColorSchemeDefault)
		}
	})
}
