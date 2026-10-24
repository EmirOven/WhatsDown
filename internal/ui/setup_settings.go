package ui

import (
	"context"
	"log"
	"time"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"go.mau.fi/whatsmeow/appstate"
	"postbetasoftware.com/whatsdown/internal/avatar"
)

var (
	OnLogoutRequested func()
	OnDemoModeToggled func(enabled bool)
)

func (ctrl *MainController) bindSettingsWidgets(builder *gtk.Builder) {
	if row, ok := castObject[*adw.ActionRow](builder, "settings_about_row"); ok {
		row.ConnectActivated(func() { ctrl.showAboutDialog() })
	}
	ctrl.settingsProfileRow, _ = castObject[*adw.ActionRow](builder, "settings_profile_row")
	ctrl.settingsProfileAvatar, _ = castObject[*adw.Avatar](builder, "settings_profile_avatar")
	ctrl.settingsResyncBtn, _ = castObject[*gtk.Button](builder, "settings_resync_btn")
	ctrl.settingsDbPathRow, _ = castObject[*adw.ActionRow](builder, "settings_db_path_row")
	ctrl.settingsStatsRow, _ = castObject[*adw.ActionRow](builder, "settings_stats_row")
	ctrl.settingsClearCacheBtn, _ = castObject[*gtk.Button](builder, "settings_clear_cache_btn")
	ctrl.settingsThemeRow, _ = castObject[*adw.ComboRow](builder, "settings_theme_row")
	ctrl.settingsTintingRow, _ = castObject[*adw.SwitchRow](builder, "settings_tinting_row")
	ctrl.settingsDemoRow, _ = castObject[*adw.SwitchRow](builder, "settings_demo_row")
	ctrl.settingsLogoutBtn, _ = castObject[*gtk.Button](builder, "settings_logout_btn")

	if ctrl.settingsResyncBtn != nil {
		ctrl.settingsResyncBtn.ConnectClicked(func() {
			ctrl.settingsResyncBtn.SetSensitive(false)
			ctrl.settingsResyncBtn.SetLabel("Syncing…")
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
				defer cancel()
				if ctrl.client != nil {
					for _, name := range appstate.AllPatchNames {
						// Request changes since the stored version instead of deleting
						// the version and replaying the complete account state.
						if err := ctrl.client.FetchAppState(ctx, name, false, false); err != nil {
							log.Printf("syncing app state %s: %v", name, err)
							break
						}
					}
				}
				if ctrl.store != nil {
					ctrl.store.ImportStickersFromChatHistory(ctx, ctrl.client)
				}
				glib.IdleAdd(func() {
					ctrl.settingsResyncBtn.SetSensitive(true)
					ctrl.settingsResyncBtn.SetLabel("Sync Now")
					ctrl.loadChats()
					ctrl.loadArchivedChats()
					ctrl.loadSavedStickersUI()
					ctrl.refreshSettingsInfo()
				})
			}()
		})
	}

	if ctrl.settingsClearCacheBtn != nil {
		ctrl.settingsClearCacheBtn.ConnectClicked(func() {
			avatar.ClearCache()
			ctrl.settingsClearCacheBtn.SetLabel("Cleared!")
			glib.TimeoutAdd(2000, func() bool {
				ctrl.settingsClearCacheBtn.SetLabel("Clear")
				return false
			})
		})
	}

	ctrl.bindThemeRow()
	ctrl.bindDonationButtons(builder)

	if ctrl.settingsLogoutBtn != nil {
		ctrl.settingsLogoutBtn.ConnectClicked(func() {
			go func() {
				if ctrl.client != nil {
					_ = ctrl.client.Logout(context.Background())
				}
				glib.IdleAdd(func() {
					activeMainWindow = nil
					if OnLogoutRequested != nil {
						OnLogoutRequested()
					}
				})
			}()
		})
	}
}
