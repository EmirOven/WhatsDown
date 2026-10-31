package app

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"postbetasoftware.com/whatsdown/internal/media"
	"postbetasoftware.com/whatsdown/internal/store"
)

func TestInstallDemoMediaAndReset(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	installDemoMedia()
	for _, asset := range store.DemoMediaAssets(time.Now()) {
		fi, err := os.Stat(media.GetLocalMediaPath(asset.Message))
		if err != nil || fi.Size() == 0 {
			t.Errorf("demo asset %s for %s was not installed: %v", asset.Asset, asset.Message.ID, err)
		}
	}

	db := filepath.Join(t.TempDir(), "whatsdown_demo.db")
	if err := os.WriteFile(db, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	resetOutdatedDemoDatabase(db)
	if _, err := os.Stat(db); !os.IsNotExist(err) {
		t.Fatalf("outdated demo database should be removed, got %v", err)
	}
	if err := os.WriteFile(db, []byte("current"), 0o600); err != nil {
		t.Fatal(err)
	}
	resetOutdatedDemoDatabase(db)
	if _, err := os.Stat(db); err != nil {
		t.Fatalf("current demo database should be kept: %v", err)
	}
}
