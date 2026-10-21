package app

import (
	"os"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"postbetasoftware.com/whatsdown/internal/store"
)

var devMode = os.Getenv("WHATSDOWN_DEV") == "1"

type page int

const (
	pageInitial page = iota // QR login
	pageSyncing             // history sync progress
	pageMain                // main application window
)

type syncState struct {
	reconnecting    bool
	percent         int
	conversations   int
	messages        int
	offlineTotal    int
	offlineReceived int
	done            bool
}

func newSyncState(reconnecting bool) syncState {
	syncTransitionScheduled = false
	return syncState{reconnecting: reconnecting, percent: -1, offlineTotal: -1}
}

// Global app state.
var (
	application              *adw.Application
	currentPage              page
	currentWindow            *adw.ApplicationWindow
	qrImage                  *gtk.Image
	qrTexture                *gdk.Texture
	syncProgressBar          *gtk.ProgressBar
	syncStatusLabel          *gtk.Label
	syncPulse                glib.SourceHandle
	syncInfo                 syncState
	waClient                 *whatsmeow.Client
	msgStore                 *store.MessageStore
	waContainer              *sqlstore.Container
	syncTransitionScheduled  bool
	cssProvider              *gtk.CSSProvider
)
