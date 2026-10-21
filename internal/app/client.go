package app

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/diamondburned/gotk4/pkg/glib/v2"
	_ "github.com/mattn/go-sqlite3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
	"postbetasoftware.com/whatsdown/internal/store"
)

func databasePath() (string, error) {
	dir := filepath.Join(glib.GetUserDataDir(), "whatsdown")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return filepath.Join(dir, "whatsdown.db"), nil
}

func newWhatsAppClient() (*whatsmeow.Client, error) {
	ctx := context.Background()
	if waContainer == nil {
		dbPath, err := databasePath()
		if err != nil {
			return nil, err
		}
		dsn := (&url.URL{Scheme: "file", Path: dbPath, RawQuery: "_foreign_keys=on&_busy_timeout=5000"}).String()
		db, err := sql.Open("sqlite3", dsn)
		if err != nil {
			return nil, fmt.Errorf("opening database: %w", err)
		}
		container := sqlstore.NewWithDB(db, "sqlite3", waLog.Stdout("Database", "INFO", true))
		if err := container.Upgrade(ctx); err != nil {
			return nil, fmt.Errorf("upgrading session store: %w", err)
		}
		waContainer = container

		s, err := store.OpenMessageStore(ctx, db)
		if err != nil {
			return nil, fmt.Errorf("opening message store: %w", err)
		}
		msgStore = s
	}

	deviceStore, err := waContainer.GetFirstDevice(ctx)
	if err != nil {
		return nil, err
	}
	client := whatsmeow.NewClient(deviceStore, waLog.Stdout("Client", "INFO", true))
	client.EmitAppStateEventsOnFullSync = true
	client.AddEventHandler(handleWhatsAppEvent)
	waClient = client
	return client, nil
}
