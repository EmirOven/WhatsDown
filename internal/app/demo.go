package app

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/store"
	"postbetasoftware.com/whatsdown/internal/ui"
)

func parseCustomArgs(args []string) []string {
	var filtered []string
	for _, a := range args {
		switch a {
		case "--demo", "--dev-demo":
			demoMode, devMode = true, true
		case "--dev":
			devMode = true
		default:
			filtered = append(filtered, a)
		}
	}
	return filtered
}

func activateDemoWindow(app *adw.Application, cssProvider *gtk.CSSProvider) {
	s, err := openDemoStore()
	if err != nil {
		log.Fatalf("opening demo store: %v", err)
	}
	msgStore = s
	if err := showPage(pageMain); err != nil {
		log.Fatalf("building demo window: %v", err)
	}
	if devMode {
		startLiveReload(app, cssProvider)
	}
}
func openDemoStore() (*store.MessageStore, error) {
	if demoMsgStore != nil {
		return demoMsgStore, nil
	}
	dir := filepath.Join(glib.GetUserDataDir(), "whatsdown")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	dbPath := filepath.Join(dir, "whatsdown_demo.db")
	resetOutdatedDemoDatabase(dbPath)
	dsn := (&url.URL{Scheme: "file", Path: dbPath, RawQuery: "_foreign_keys=on&_journal_mode=WAL&_busy_timeout=30000&_synchronous=NORMAL&_txlock=immediate"}).String()
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening demo database: %w", err)
	}
	ctx := context.Background()
	s, err := store.OpenMessageStore(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("opening demo message store: %w", err)
	}
	_ = s.SeedDemoData(ctx)
	installDemoMedia()
	demoMsgStore = s
	return s, nil
}

func toggleDemoMode(enabled bool) {
	demoMode = enabled
	if enabled {
		if s, err := openDemoStore(); err == nil {
			msgStore = s
			glib.IdleAdd(func() { _ = showPage(pageMain) })
		}
		return
	}
	if realMsgStore == nil {
		if _, err := newWhatsAppClient(); err != nil {
			return
		}
	}
	msgStore = realMsgStore
	glib.IdleAdd(func() {
		if waClient != nil && waClient.Store != nil && waClient.Store.ID != nil {
			_ = showPage(pageMain)
			return
		}
		ui.SetActiveMainWindow(nil)
		_ = showPage(pageInitial)
		if waClient != nil && !waClient.IsConnected() {
			go connectWhatsApp(waClient)
		}
	})
}
