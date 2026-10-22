package ui

import (
	"context"
	"database/sql"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	_ "github.com/mattn/go-sqlite3"
	"postbetasoftware.com/whatsdown/internal/store"
)

func TestSyncDoesNotPopulateHiddenStickerPicker(t *testing.T) {
	if os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
		t.Skip("no display available")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	gtk.Init()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s, err := store.OpenMessageStore(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "sticker.png")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, image.NewRGBA(image.Rect(0, 0, 512, 512))); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	sticker := store.SavedSticker{ID: "test", FileHash: "hash", FilePath: path, Width: 512, Height: 512}
	if err := s.SaveSticker(context.Background(), sticker); err != nil {
		t.Fatal(err)
	}
	c := &MainController{store: s}
	c.stickerFlowbox = gtk.NewFlowBox()
	c.stickerPopover = gtk.NewPopover()
	c.LoadSavedStickersUI()
	if !c.savedStickerHashes["hash"] {
		t.Fatal("sync did not update sticker metadata")
	}
	if c.stickerFlowbox.FirstChild() != nil {
		t.Fatal("sync created image widgets in a hidden picker")
	}

	// When displayed, decode at the actual picker size instead of allocating
	// a 512x512 texture for every 64x64 cell.
	window := gtk.NewWindow()
	button := gtk.NewMenuButton()
	button.SetPopover(c.stickerPopover)
	window.SetChild(button)
	window.Present()
	defer window.Destroy()
	c.stickerPopover.ConnectShow(func() { c.LoadSavedStickersUI() })
	c.stickerPopover.Popup()
	var textures int
	var inspect func(gtk.Widgetter)
	inspect = func(widget gtk.Widgetter) {
		if pic, ok := widget.(*gtk.Picture); ok {
			tex := pic.Paintable()
			if tex == nil {
				t.Fatal("missing sticker preview")
			}
			if tex.IntrinsicWidth() != 64 || tex.IntrinsicHeight() != 64 {
				t.Fatalf("sticker preview size = %dx%d", tex.IntrinsicWidth(), tex.IntrinsicHeight())
			}
			textures++
		}
		for child := gtk.BaseWidget(widget).FirstChild(); child != nil; child = gtk.BaseWidget(child).NextSibling() {
			inspect(child)
		}
	}
	inspect(c.stickerFlowbox)
	if textures != 1 {
		t.Fatalf("found %d sticker previews", textures)
	}
	c.stickerPopover.Popdown()
	c.LoadSavedStickersUI()
	if c.stickerFlowbox.FirstChild() != nil || len(c.widgetSignalCleanups) != 0 {
		t.Fatal("refresh retained hidden sticker widgets or callbacks")
	}
	runtime.KeepAlive(c)
}
