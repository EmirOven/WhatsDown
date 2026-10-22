package ui

import (
	"context"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
)

func (ctrl *MainController) bindThemeRow() {
	if ctrl.settingsThemeRow != nil {
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

	ctrl.bindTintingRow()
}

func (ctrl *MainController) bindTintingRow() {
	if ctrl.settingsTintingRow == nil {
		return
	}
	if ctrl.store != nil {
		val, _ := ctrl.store.GetAppSetting(context.Background(), "tinting_enabled", "false")
		ctrl.tintingEnabled = val == "true"
	}
	ctrl.settingsTintingRow.SetActive(ctrl.tintingEnabled)

	ctrl.settingsTintingRow.Connect("notify::active", func() {
		ctrl.tintingEnabled = ctrl.settingsTintingRow.Active()
		if ctrl.store != nil {
			strVal := "false"
			if ctrl.tintingEnabled {
				strVal = "true"
			}
			_ = ctrl.store.SetAppSetting(context.Background(), "tinting_enabled", strVal)
		}
		if ctrl.currentChat != nil {
			ctrl.applyChatWallpaper(ctrl.currentChat.JID)
		}
	})
}

