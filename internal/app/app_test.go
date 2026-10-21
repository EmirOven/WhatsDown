package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/ui"
)

func initTestAssets() {
	if ui.GetAssets().InitialPopupUI != "" {
		return
	}
	read := func(rel string) string {
		data, _ := os.ReadFile(filepath.Join("../..", rel))
		return string(data)
	}
	ui.SetEmbeddedAssets(ui.Assets{
		InitialPopupUI:  read("ui/WhatsDownInitial.ui"),
		InitialPopupCSS: read("ui/WhatsDownInitial.css"),
		SyncingUI:       read("ui/WhatsDownSyncing.ui"),
		SyncingCSS:      read("ui/WhatsDownSyncing.css"),
		MainUI:          read("ui/WhatsDownMain.ui"),
		MainCSS:         read("ui/WhatsDownMain.css"),
		ChatsMenuUI:     read("ui/components/ChatsMenu.ui"),
		CallsMenuUI:     read("ui/components/CallsMenu.ui"),
		ArchivedMenuUI:  read("ui/components/ArchivedMenu.ui"),
		SettingsMenuUI:  read("ui/components/SettingsMenu.ui"),
		RightChatViewUI: read("ui/components/RightChatView.ui"),
	})
}

func TestBuildPages(t *testing.T) {
	if os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
		t.Skip("no display available")
	}
	gtk.Init()
	initTestAssets()

	for _, p := range []page{pageInitial, pageSyncing, pageMain} {
		xml := p.embeddedUI()
		if xml == "" {
			t.Fatalf("embedded UI empty for page %v", p)
		}
		builder := gtk.NewBuilder()
		if err := builder.AddFromString(xml); err != nil {
			t.Fatalf("builder.AddFromString failed for page %v: %v", p, err)
		}
		if p == pageMain {
			mergeComponents(builder)
			w, err := ui.SetupMainWindow(builder, nil, nil)
			if err != nil {
				t.Fatalf("SetupMainWindow failed: %v", err)
			}
			if w == nil {
				t.Fatalf("SetupMainWindow returned nil window")
			}
		} else if p == pageSyncing {
			w, ok1 := castObject[*adw.ApplicationWindow](builder, "syncing_window")
			bar, ok2 := castObject[*gtk.ProgressBar](builder, "sync_progress")
			status, ok3 := castObject[*gtk.Label](builder, "sync_status")
			if !ok1 || !ok2 || !ok3 || w == nil || bar == nil || status == nil {
				t.Fatalf("syncing page widgets missing: %v %v %v", ok1, ok2, ok3)
			}
		} else if p == pageInitial {
			w, ok1 := castObject[*adw.ApplicationWindow](builder, "initial_window")
			img, ok2 := castObject[*gtk.Image](builder, "qr_image")
			if !ok1 || !ok2 || w == nil || img == nil {
				t.Fatalf("initial page widgets missing: %v %v", ok1, ok2)
			}
		}
	}
}
