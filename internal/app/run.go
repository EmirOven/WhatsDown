package app

import (
	"log"
	"os"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"postbetasoftware.com/whatsdown/internal/ui"
)

// Run starts the WhatsDown GTK/Adwaita application.
func Run() {
	gtkArgs := parseCustomArgs(os.Args)
	flags := gio.ApplicationDefaultFlags
	if devMode {
		flags = gio.ApplicationNonUnique
	}
	app := adw.NewApplication("com.postbetasoftware.WhatsDown", flags)
	application = app
	ui.SetAppInstance(&app.Application.Application)
	ui.OnDemoModeToggled = toggleDemoMode

	ui.OnLogoutRequested = func() {
		glib.IdleAdd(func() {
			ui.SetActiveMainWindow(nil)
			syncInfo = newSyncState(false)
			qrTexture = nil
			if err := showPage(pageInitial); err != nil {
				log.Printf("showing initial page: %v", err)
			}
			go func() {
				client, err := newWhatsAppClient()
				if err != nil {
					log.Printf("opening WhatsApp session store: %v", err)
					return
				}
				connectWhatsApp(client)
			}()
		})
	}

	app.ConnectActivate(func() {
		if currentWindow != nil {
			currentWindow.Present()
			return
		}

		cssProvider := newCSSProvider()
		loadCSS(cssProvider)

		if demoMode {
			activateDemoWindow(app, cssProvider)
			return
		}

		client, err := newWhatsAppClient()
		if err != nil {
			log.Fatalf("opening WhatsApp session store: %v", err)
		}
		loggedIn := client.Store.ID != nil
		syncInfo = newSyncState(loggedIn)

		start := pageInitial
		if loggedIn {
			start = pageSyncing
		}
		devPreview := false
		if devMode {
			switch os.Getenv("WHATSDOWN_DEV_PAGE") {
			case "syncing":
				start = pageSyncing
				devPreview = true
			case "main":
				start = pageMain
				devPreview = true
			}
		}

		if err := showPage(start); err != nil {
			log.Fatalf("building window: %v", err)
		}

		if devMode {
			startLiveReload(app, cssProvider)
		}

		if !devPreview {
			go connectWhatsApp(client)
		}
	})

	if code := app.Run(gtkArgs); code > 0 {
		os.Exit(code)
	}
}
