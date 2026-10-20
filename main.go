package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	_ "github.com/mattn/go-sqlite3"
	"github.com/skip2/go-qrcode"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/appstate"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
)

//go:embed ui/WhatsDownInitial.ui
var InitialPopupUI string

//go:embed ui/WhatsDownInitial.css
var InitialPopupCSS string

//go:embed ui/WhatsDownSyncing.ui
var SyncingUI string

//go:embed ui/WhatsDownSyncing.css
var SyncingCSS string

//go:embed ui/WhatsDownMain.ui
var MainUI string

//go:embed ui/WhatsDownMain.css
var MainCSS string

//go:embed ui/components/ChatsMenu.ui
var ChatsMenuUI string

//go:embed ui/components/CallsMenu.ui
var CallsMenuUI string

//go:embed ui/components/ArchivedMenu.ui
var ArchivedMenuUI string

//go:embed ui/components/SettingsMenu.ui
var SettingsMenuUI string

//go:embed ui/components/RightChatView.ui
var RightChatViewUI string

// On-disk asset paths, used instead of the embedded copies in dev mode.
const (
	uiDir          = "ui"
	initialUIPath  = "ui/WhatsDownInitial.ui"
	initialCSSPath = "ui/WhatsDownInitial.css"
	syncingUIPath  = "ui/WhatsDownSyncing.ui"
	syncingCSSPath = "ui/WhatsDownSyncing.css"
	mainUIPath     = "ui/WhatsDownMain.ui"
	mainCSSPath    = "ui/WhatsDownMain.css"
)

// page identifies which window is currently shown.
type page int

const (
	pageInitial page = iota // QR login
	pageSyncing             // history sync progress
	pageMain                // main application window
)

func (p page) uiPath() string {
	switch p {
	case pageSyncing:
		return syncingUIPath
	case pageMain:
		return mainUIPath
	default:
		return initialUIPath
	}
}

func (p page) embeddedUI() string {
	switch p {
	case pageSyncing:
		return SyncingUI
	case pageMain:
		return MainUI
	default:
		return InitialPopupUI
	}
}

var devMode = os.Getenv("WHATSDOWN_DEV") == "1"

// UI state. Only touched from the GTK main thread.
var (
	application   *adw.Application
	currentPage   page
	currentWindow *adw.ApplicationWindow

	// Initial page widgets.
	qrImage *gtk.Image
	// Latest QR code, kept so it survives window rebuilds during live reload.
	qrTexture *gdk.Texture

	// Syncing page widgets.
	syncProgressBar *gtk.ProgressBar
	syncStatusLabel *gtk.Label
	syncPulse       glib.SourceHandle
	// Sync progress, kept so it survives window rebuilds during live reload.
	syncInfo syncState

	// WhatsApp client and message storage.
	waClient *whatsmeow.Client
	msgStore *messageStore

	syncTransitionScheduled bool
)

type syncState struct {
	// reconnecting is true when resuming an existing session (fetching
	// messages missed while offline) rather than syncing after a QR scan.
	reconnecting bool

	// First login: history sync pushed by the phone.
	percent       int // last reported progress, -1 if unknown
	conversations int
	messages      int

	// Reconnect: messages the server queued while we were offline.
	offlineTotal    int // -1 until the server sends a preview
	offlineReceived int

	done bool
}

func newSyncState(reconnecting bool) syncState {
	syncTransitionScheduled = false
	return syncState{reconnecting: reconnecting, percent: -1, offlineTotal: -1}
}

func main() {
	app := adw.NewApplication("com.postbetasoftware.WhatsDown", gio.ApplicationDefaultFlags)
	application = app

	app.ConnectActivate(func() {
		if currentWindow != nil {
			currentWindow.Present()
			return
		}

		cssProvider := newCSSProvider()
		loadCSS(cssProvider)

		// Opening the store is a quick local SQLite read; it tells us whether
		// a device is already paired, which decides the first page.
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
		// WHATSDOWN_DEV_PAGE opens a specific page directly (syncing or main),
		// so its layout can be worked on with live reload without scanning a QR code.
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

	if code := app.Run(os.Args); code > 0 {
		os.Exit(code)
	}
}

func newCSSProvider() *gtk.CSSProvider {
	provider := gtk.NewCSSProvider()
	gtk.StyleContextAddProviderForDisplay(
		gdk.DisplayGetDefault(),
		provider,
		gtk.STYLE_PROVIDER_PRIORITY_APPLICATION,
	)
	return provider
}

// loadCSS loads the stylesheets of all pages into the provider, from disk in
// dev mode and from the embedded copies otherwise.
func loadCSS(provider *gtk.CSSProvider) {
	css := InitialPopupCSS + "\n" + SyncingCSS + "\n" + MainCSS
	if devMode {
		css = ""
		for _, path := range []string{initialCSSPath, syncingCSSPath, mainCSSPath} {
			data, err := os.ReadFile(path)
			if err != nil {
				log.Printf("reading %s: %v", path, err)
				continue
			}
			css += string(data) + "\n"
		}
	}
	provider.LoadFromString(css)
}

// showPage builds the window for p and presents it, replacing the current
// window. On error the current window is kept.
func showPage(p page) error {
	uiXML := p.embeddedUI()
	if devMode {
		data, err := os.ReadFile(p.uiPath())
		if err != nil {
			return fmt.Errorf("reading %s: %w", p.uiPath(), err)
		}
		uiXML = string(data)
	}
	return buildWindow(p, uiXML)
}

// buildWindow builds the window for p from UI XML and presents it.
// If a window already exists it is replaced; on error the old one is kept.
func buildWindow(p page, uiXML string) error {
	builder := gtk.NewBuilder()
	if err := builder.AddFromString(uiXML); err != nil {
		destroyBuilderWindows(builder)
		return err
	}

	if p == pageMain {
		components := []struct {
			path     string
			embedded string
		}{
			{"ui/components/ChatsMenu.ui", ChatsMenuUI},
			{"ui/components/CallsMenu.ui", CallsMenuUI},
			{"ui/components/ArchivedMenu.ui", ArchivedMenuUI},
			{"ui/components/SettingsMenu.ui", SettingsMenuUI},
			{"ui/components/RightChatView.ui", RightChatViewUI},
		}
		for _, c := range components {
			xml := c.embedded
			if devMode {
				if data, err := os.ReadFile(c.path); err == nil {
					xml = string(data)
				}
			}
			if err := builder.AddFromString(xml); err != nil {
				log.Printf("merging component %s: %v", c.path, err)
			}
		}
	}

	var window *adw.ApplicationWindow
	switch p {
	case pageInitial:
		w, ok1 := castObject[*adw.ApplicationWindow](builder, "initial_window")
		image, ok2 := castObject[*gtk.Image](builder, "qr_image")
		if !ok1 || !ok2 {
			destroyBuilderWindows(builder)
			return errors.New("UI must define an Adw.ApplicationWindow 'initial_window' and a Gtk.Image 'qr_image'")
		}
		if qrTexture != nil {
			image.SetFromPaintable(qrTexture)
		}
		window = w
		qrImage = image
		syncProgressBar, syncStatusLabel = nil, nil

	case pageSyncing:
		w, ok1 := castObject[*adw.ApplicationWindow](builder, "syncing_window")
		bar, ok2 := castObject[*gtk.ProgressBar](builder, "sync_progress")
		status, ok3 := castObject[*gtk.Label](builder, "sync_status")
		if !ok1 || !ok2 || !ok3 {
			destroyBuilderWindows(builder)
			return errors.New("UI must define an Adw.ApplicationWindow 'syncing_window', a Gtk.ProgressBar 'sync_progress' and a Gtk.Label 'sync_status'")
		}
		window = w
		qrImage = nil
		syncProgressBar, syncStatusLabel = bar, status
		renderSync()

	case pageMain:
		w, err := setupMainWindow(builder)
		if err != nil {
			destroyBuilderWindows(builder)
			return err
		}
		window = w
		qrImage = nil
		syncProgressBar, syncStatusLabel = nil, nil
	}

	// Present the new window before destroying the old one so the
	// application never has zero windows (which would make it quit).
	window.SetApplication(&application.Application)
	window.Present()

	old := currentWindow
	currentWindow, currentPage = window, p
	if old != nil {
		old.Destroy()
	}
	return nil
}

func castObject[T any](builder *gtk.Builder, id string) (T, bool) {
	var zero T
	obj := builder.GetObject(id)
	if obj == nil {
		return zero, false
	}
	v, ok := obj.Cast().(T)
	return v, ok
}

// destroyBuilderWindows cleans up half-built toplevels after a failed build.
func destroyBuilderWindows(builder *gtk.Builder) {
	for _, id := range []string{"initial_window", "syncing_window", "main_window"} {
		if obj := builder.GetObject(id); obj != nil {
			if w, ok := obj.Cast().(interface{ Destroy() }); ok {
				w.Destroy()
			}
		}
	}
}

// setQRTexture must be called on the GTK main thread.
func setQRTexture(texture *gdk.Texture) {
	qrTexture = texture
	if qrImage != nil {
		qrImage.SetFromPaintable(texture)
	}
}

// renderSync shows the current sync state on the syncing page. While no
// progress figure is known yet the progress bar pulses.
// Must be called on the GTK main thread.
func renderSync() {
	if syncProgressBar == nil {
		return
	}

	s := syncInfo
	var fraction float64 = -1 // -1 = unknown, pulse
	var barText, status string

	switch {
	case s.reconnecting && s.done:
		fraction, barText = 1, "Up to date"
		switch s.offlineReceived {
		case 0:
			status = "No new messages."
		case 1:
			status = "Received 1 new message."
		default:
			status = fmt.Sprintf("Received %d new messages.", s.offlineReceived)
		}
	case s.reconnecting && s.offlineTotal > 0:
		received := min(s.offlineReceived, s.offlineTotal)
		fraction = float64(received) / float64(s.offlineTotal)
		barText = fmt.Sprintf("%d / %d", received, s.offlineTotal)
		status = "Fetching messages received while WhatsDown was closed…"
	case s.reconnecting:
		barText = "Connecting…"
		status = "Checking for new messages."

	case s.done:
		fraction, barText = 1, "Done"
		status = fmt.Sprintf("Synced %d chats and %d messages.", s.conversations, s.messages)
	case s.percent >= 0:
		fraction = float64(s.percent) / 100
		barText = fmt.Sprintf("%d%%", s.percent)
		status = fmt.Sprintf("%d chats and %d messages received so far…", s.conversations, s.messages)
	default:
		barText = "Waiting for your phone…"
		status = "Your phone is preparing your chat history."
	}

	syncProgressBar.SetText(barText)
	syncStatusLabel.SetText(status)
	if fraction >= 0 {
		syncProgressBar.SetFraction(fraction)
	}

	pulsing := fraction < 0
	if pulsing && syncPulse == 0 {
		syncPulse = glib.TimeoutAdd(120, func() bool {
			if syncProgressBar == nil {
				syncPulse = 0
				return false
			}
			syncProgressBar.Pulse()
			return true
		})
	} else if !pulsing && syncPulse != 0 {
		glib.SourceRemove(syncPulse)
		syncPulse = 0
	}

	// Once syncing completes, open WhatsDownMain
	if s.done && !syncTransitionScheduled {
		syncTransitionScheduled = true
		glib.TimeoutAdd(900, func() bool {
			if currentPage == pageSyncing && syncInfo.done {
				if err := showPage(pageMain); err != nil {
					log.Printf("showing main page: %v", err)
				}
			}
			return false
		})
	}
}

// handleHistorySync records a history sync chunk. Must be called on the GTK
// main thread.
func handleHistorySync(conversations, messages int, percent int) {
	syncInfo.conversations += conversations
	syncInfo.messages += messages
	if percent >= 0 {
		syncInfo.percent = percent
		if percent >= 100 {
			syncInfo.done = true
		}
	}
	renderSync()
}

// databasePath returns the session database location inside the XDG user
// data directory (usually ~/.local/share/whatsdown/whatsdown.db), creating
// the directory if needed.
func databasePath() (string, error) {
	dir := filepath.Join(glib.GetUserDataDir(), "whatsdown")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return filepath.Join(dir, "whatsdown.db"), nil
}

// The session store, opened once and shared by every client we create.
var waContainer *sqlstore.Container

// newWhatsAppClient opens the session store (on first use) and returns a
// client for the stored device. If nothing is paired yet the device is new and
// client.Store.ID is nil.
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

		s, err := openMessageStore(ctx, db)
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

// connectWhatsApp connects the client. A paired client just reconnects (the
// server then sends whatever was missed while offline); an unpaired one goes
// through QR login.
func connectWhatsApp(client *whatsmeow.Client) {
	if client.Store.ID != nil {
		if err := client.Connect(); err != nil {
			log.Printf("connecting to WhatsApp: %v", err)
		} else {
			go func() {
				time.Sleep(500 * time.Millisecond)
				for _, name := range appstate.AllPatchNames {
					_ = client.FetchAppState(context.Background(), name, false, false)
				}
			}()
		}
		return
	}

	qrChan, err := client.GetQRChannel(context.Background())
	if err != nil {
		log.Printf("starting QR login: %v", err)
		return
	}

	if err := client.Connect(); err != nil {
		log.Printf("connecting to WhatsApp: %v", err)
		return
	}

	for evt := range qrChan {
		if evt.Event == whatsmeow.QRChannelEventCode {
			pngBytes, err := qrcode.Encode(evt.Code, qrcode.Medium, 256)
			if err != nil {
				continue
			}

			glib.IdleAdd(func() {
				texture, err := gdk.NewTextureFromBytes(glib.NewBytesWithGo(pngBytes))
				if err == nil {
					setQRTexture(texture)
				}
			})
		}
	}
}

// handleWhatsAppEvent runs on whatsmeow's goroutines; UI work is handed off to
// the GTK main thread.
func handleWhatsAppEvent(evt any) {
	switch e := evt.(type) {
	case *events.PairSuccess:
		// The QR code was scanned: switch to the syncing page.
		glib.IdleAdd(func() {
			syncInfo = newSyncState(false)
			if currentPage != pageSyncing {
				if err := showPage(pageSyncing); err != nil {
					log.Printf("showing syncing page: %v", err)
				}
			}
		})

	case *events.HistorySync:
		conversations := e.Data.GetConversations()
		messages := 0
		for _, conv := range conversations {
			messages += len(conv.GetMessages())
		}
		percent := -1
		if e.Data.Progress != nil {
			percent = int(e.Data.GetProgress())
		}
		glib.IdleAdd(func() {
			handleHistorySync(len(conversations), messages, percent)
		})
		go func() {
			if msgStore != nil && waClient != nil {
				stored, err := msgStore.saveHistorySync(context.Background(), waClient, e.Data)
				if err != nil {
					log.Printf("saving history sync: %v", err)
				} else {
					log.Printf("saved %d messages from history sync", stored)
				}
				glib.IdleAdd(func() {
					if activeMainWindow != nil {
						activeMainWindow.loadChats()
						activeMainWindow.loadArchivedChats()
						activeMainWindow.loadSavedStickersUI()
					}
				})
			}
		}()

	case *events.OfflineSyncPreview:
		// Sent right after reconnecting when messages are waiting for us.
		total := e.Messages
		glib.IdleAdd(func() {
			syncInfo.offlineTotal = total
			renderSync()
		})

	case *events.Message:
		go func() {
			if e.Message != nil && waClient != nil && msgStore != nil {
				var rm *waE2E.ReactionMessage
				if e.Message.GetReactionMessage() != nil {
					rm = e.Message.GetReactionMessage()
				} else if e.Message.GetEncReactionMessage() != nil {
					rm, _ = waClient.DecryptReaction(context.Background(), e)
				}
				if rm != nil {
					handleIncomingReaction(e, rm)
					return
				}
			}
			if e.Message != nil && e.Message.GetPollUpdateMessage() != nil && waClient != nil && msgStore != nil {
				handleIncomingPollUpdate(e)
			}
			if msgStore != nil {
				if err := msgStore.saveLiveMessage(context.Background(), e); err != nil {
					log.Printf("saving live message: %v", err)
				}
				if e.Info.IsFromMe && e.Message != nil && e.Message.GetStickerMessage() != nil {
					go msgStore.importStickersFromChatHistory(context.Background(), waClient)
				}
			}
			glib.IdleAdd(func() {
				if syncInfo.reconnecting && !syncInfo.done {
					syncInfo.offlineReceived++
					renderSync()
				}
				if activeMainWindow != nil {
					activeMainWindow.handleIncomingLiveMessage(e)
				}
			})
		}()

	case *events.Receipt:
		status := MessageStatusNone
		switch e.Type {
		case types.ReceiptTypeRead, types.ReceiptTypeReadSelf, types.ReceiptTypePlayed, types.ReceiptTypePlayedSelf:
			status = MessageStatusRead
		case types.ReceiptTypeDelivered, types.ReceiptTypeSender:
			status = MessageStatusDelivered
		}

		if e.Type == types.ReceiptTypeRead || e.Type == types.ReceiptTypeReadSelf {
			go func() {
				if msgStore != nil {
					_ = msgStore.markChatRead(context.Background(), e.Chat)
					glib.IdleAdd(func() {
						if activeMainWindow != nil {
							activeMainWindow.loadChats()
							activeMainWindow.loadArchivedChats()
						}
					})
				}
			}()
		}

		if status > MessageStatusNone && len(e.MessageIDs) > 0 {
			msgIDs := make([]string, len(e.MessageIDs))
			for i, mid := range e.MessageIDs {
				msgIDs[i] = string(mid)
			}
			go func() {
				if msgStore != nil {
					_ = msgStore.updateMessageStatuses(context.Background(), msgIDs, status)
				}
				glib.IdleAdd(func() {
					if activeMainWindow != nil {
						activeMainWindow.handleReceiptUpdate(msgIDs, status)
					}
				})
			}()
		}

	case *events.Picture:
		jidStr := e.JID.ToNonAD().String()
		invalidateAvatar(jidStr)
		glib.IdleAdd(func() {
			if activeMainWindow != nil {
				activeMainWindow.loadChats()
				if activeMainWindow.currentChat != nil && activeMainWindow.currentChat.JID == jidStr {
					loadAvatar(jidStr, activeMainWindow.currentChat.Name, activeMainWindow.activeChatAvatar)
				}
			}
		})

	case *events.Contact, *events.PushName, *events.BusinessName:
		glib.IdleAdd(func() {
			if activeMainWindow != nil {
				activeMainWindow.loadChats()
			}
		})

	case *events.Connected:
		go func() {
			time.Sleep(500 * time.Millisecond)
			for _, name := range appstate.AllPatchNames {
				_ = waClient.FetchAppState(context.Background(), name, false, false)
			}
			if msgStore != nil && waClient != nil {
				msgStore.importStickersFromChatHistory(context.Background(), waClient)
				groups, err := waClient.GetJoinedGroups(context.Background())
				if err == nil && len(groups) > 0 {
					_ = msgStore.syncGroups(context.Background(), groups)
					glib.IdleAdd(func() {
						if activeMainWindow != nil {
							activeMainWindow.loadChats()
						}
					})
				}
			}
		}()

	case *events.CallOffer:
		log.Printf("incoming call offer: callID=%s from=%s caller=%s", e.CallID, e.From, e.CallCreator)
		if msgStore != nil {
			callerJID := e.CallCreator.ToNonAD().String()
			if callerJID == "" {
				callerJID = e.From.ToNonAD().String()
			}
			chatJID := e.From.ToNonAD().String()
			if chatJID == "" {
				chatJID = callerJID
			}
			go func() {
				_ = msgStore.saveCall(context.Background(), CallRecord{
					ID:        e.CallID,
					ChatJID:   chatJID,
					CallerJID: callerJID,
					Timestamp: e.Timestamp,
					Outcome:   CallOutcomeConnected,
					FromMe:    false,
				})
				glib.IdleAdd(func() {
					if activeMainWindow != nil {
						activeMainWindow.loadCalls()
					}
				})
			}()
		}

	case *events.CallTerminate:
		log.Printf("call terminate: callID=%s reason=%s", e.CallID, e.Reason)
		if msgStore != nil {
			outcome := CallOutcomeConnected
			rLower := strings.ToLower(e.Reason)
			if strings.Contains(rLower, "miss") || strings.Contains(rLower, "timeout") {
				outcome = CallOutcomeMissed
			} else if strings.Contains(rLower, "reject") || strings.Contains(rLower, "decline") {
				outcome = CallOutcomeRejected
			}
			callerJID := e.CallCreator.ToNonAD().String()
			if callerJID == "" {
				callerJID = e.From.ToNonAD().String()
			}
			chatJID := e.From.ToNonAD().String()
			if chatJID == "" {
				chatJID = callerJID
			}
			go func() {
				_ = msgStore.saveCall(context.Background(), CallRecord{
					ID:        e.CallID,
					ChatJID:   chatJID,
					CallerJID: callerJID,
					Timestamp: e.Timestamp,
					Outcome:   outcome,
					FromMe:    false,
				})
				glib.IdleAdd(func() {
					if activeMainWindow != nil {
						activeMainWindow.loadCalls()
					}
				})
			}()
		}

	case *events.GroupInfo:
		if waClient != nil && msgStore != nil {
			go func() {
				groups, err := waClient.GetJoinedGroups(context.Background())
				if err == nil {
					_ = msgStore.syncGroups(context.Background(), groups)
					glib.IdleAdd(func() {
						if activeMainWindow != nil {
							activeMainWindow.loadChats()
						}
					})
				}
			}()
		}

	case *events.Pin:
		if e.Action != nil && msgStore != nil {
			chatJID := e.JID.ToNonAD().String()
			pinned := e.Action.GetPinned()
			go func() {
				_ = msgStore.setChatPinned(context.Background(), chatJID, pinned)
				glib.IdleAdd(func() {
					if activeMainWindow != nil {
						activeMainWindow.loadChats()
						activeMainWindow.loadArchivedChats()
					}
				})
			}()
		}

	case *events.Archive:
		if e.Action != nil && msgStore != nil {
			chatJID := e.JID.ToNonAD().String()
			archived := e.Action.GetArchived()
			go func() {
				_ = msgStore.setChatArchived(context.Background(), chatJID, archived)
				glib.IdleAdd(func() {
					if activeMainWindow != nil {
						activeMainWindow.loadChats()
						activeMainWindow.loadArchivedChats()
					}
				})
			}()
		}

	case *events.Mute:
		if e.Action != nil && msgStore != nil {
			chatJID := e.JID.ToNonAD().String()
			muted := e.Action.GetMuted()
			go func() {
				_ = msgStore.setChatMuted(context.Background(), chatJID, muted)
				glib.IdleAdd(func() {
					if activeMainWindow != nil {
						activeMainWindow.loadChats()
						activeMainWindow.loadArchivedChats()
					}
				})
			}()
		}

	case *events.Star:
		if e.Action != nil && msgStore != nil {
			chatJID := e.ChatJID.ToNonAD().String()
			starred := e.Action.GetStarred()
			msgID := e.MessageID
			go func() {
				ctx := context.Background()
				_ = msgStore.toggleMessageStarred(ctx, chatJID, msgID, starred)
				glib.IdleAdd(func() {
					if activeMainWindow != nil && activeMainWindow.currentChat != nil {
						if activeMainWindow.isSameChat(activeMainWindow.currentChat.JID, chatJID) {
							activeMainWindow.updateSingleMessageRow(msgID)
						}
					}
				})
			}()
		}

	case *events.AppStateSyncComplete:
		glib.IdleAdd(func() {
			if activeMainWindow != nil {
				activeMainWindow.loadChats()
				activeMainWindow.loadArchivedChats()
			}
		})

	case *events.OfflineSyncCompleted:
		// Always sent after reconnecting, even when nothing was missed.
		glib.IdleAdd(func() {
			if syncInfo.reconnecting {
				syncInfo.done = true
				renderSync()
			}
		})

	case *events.LoggedOut:
		// The session was revoked from the phone: back to QR login.
		log.Printf("logged out by WhatsApp (reason %v), showing QR login", e.Reason)
		glib.IdleAdd(func() {
			activeMainWindow = nil
			syncInfo = newSyncState(false)
			qrTexture = nil
			if err := showPage(pageInitial); err != nil {
				log.Printf("showing QR page: %v", err)
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
}

// ensurePollMessageSecret ensures that the poll's 32-byte message secret is stored in
// waClient.Store.MsgSecrets for chatJID/senderJID combinations (including LID and PN variants).
func ensurePollMessageSecret(ctx context.Context, msg StoredMessage) {
	if waClient == nil || waClient.Store == nil || waClient.Store.MsgSecrets == nil || len(msg.Raw) == 0 {
		return
	}
	waMsg := parseE2EMessage(msg.Raw)
	if waMsg == nil {
		return
	}
	msgSecret := waMsg.GetMessageContextInfo().GetMessageSecret()
	if len(msgSecret) == 0 {
		return
	}

	chatJID, _ := types.ParseJID(msg.ChatJID)
	senderJID, _ := types.ParseJID(msg.SenderJID)
	if senderJID.IsEmpty() && waClient.Store.ID != nil {
		senderJID = waClient.Store.ID.ToNonAD()
	}

	// Put for the direct chat and sender JIDs
	if !chatJID.IsEmpty() && !senderJID.IsEmpty() {
		_ = waClient.Store.MsgSecrets.PutMessageSecret(ctx, chatJID.ToNonAD(), senderJID.ToNonAD(), msg.ID, msgSecret)
	}

	// Also store for LID / PN alternates if available in store
	if waClient.Store.LIDs != nil {
		if altChat, err := waClient.Store.LIDs.GetLIDForPN(ctx, chatJID.ToNonAD()); err == nil && !altChat.IsEmpty() {
			_ = waClient.Store.MsgSecrets.PutMessageSecret(ctx, altChat.ToNonAD(), senderJID.ToNonAD(), msg.ID, msgSecret)
		}
		if altSender, err := waClient.Store.LIDs.GetLIDForPN(ctx, senderJID.ToNonAD()); err == nil && !altSender.IsEmpty() {
			_ = waClient.Store.MsgSecrets.PutMessageSecret(ctx, chatJID.ToNonAD(), altSender.ToNonAD(), msg.ID, msgSecret)
		}
		if altChat, err := waClient.Store.LIDs.GetPNForLID(ctx, chatJID.ToNonAD()); err == nil && !altChat.IsEmpty() {
			_ = waClient.Store.MsgSecrets.PutMessageSecret(ctx, altChat.ToNonAD(), senderJID.ToNonAD(), msg.ID, msgSecret)
		}
		if altSender, err := waClient.Store.LIDs.GetPNForLID(ctx, senderJID.ToNonAD()); err == nil && !altSender.IsEmpty() {
			_ = waClient.Store.MsgSecrets.PutMessageSecret(ctx, chatJID.ToNonAD(), altSender.ToNonAD(), msg.ID, msgSecret)
		}
	}

	// If self chat, also store with own ID / LID
	if waClient.Store.ID != nil {
		ownPN := waClient.Store.ID.ToNonAD()
		_ = waClient.Store.MsgSecrets.PutMessageSecret(ctx, ownPN, ownPN, msg.ID, msgSecret)
		if !waClient.Store.LID.IsEmpty() {
			ownLID := waClient.Store.LID.ToNonAD()
			_ = waClient.Store.MsgSecrets.PutMessageSecret(ctx, ownLID, ownLID, msg.ID, msgSecret)
			_ = waClient.Store.MsgSecrets.PutMessageSecret(ctx, ownPN, ownLID, msg.ID, msgSecret)
			_ = waClient.Store.MsgSecrets.PutMessageSecret(ctx, ownLID, ownPN, msg.ID, msgSecret)
		}
	}
}

// handleIncomingPollUpdate decrypts and records an incoming poll vote from WhatsApp.
func handleIncomingPollUpdate(evt *events.Message) {
	if waClient == nil || msgStore == nil || evt.Message == nil {
		return
	}
	pollUpdate := evt.Message.GetPollUpdateMessage()
	if pollUpdate == nil {
		return
	}
	creationKey := pollUpdate.GetPollCreationMessageKey()
	if creationKey == nil {
		return
	}
	pollID := creationKey.GetID()
	chatJID := evt.Info.Chat.ToNonAD().String()
	voterJID := evt.Info.Sender.ToNonAD().String()
	if evt.Info.IsFromMe && voterJID == "" && waClient.Store != nil && waClient.Store.ID != nil {
		voterJID = waClient.Store.ID.ToNonAD().String()
	}

	origMsg, err := msgStore.getMessage(context.Background(), chatJID, pollID)
	if err != nil || origMsg == nil {
		origMsg, err = msgStore.getMessageByID(context.Background(), pollID)
	}
	if err != nil || origMsg == nil {
		log.Printf("could not find original poll message %s in chat %s: %v", pollID, chatJID, err)
		return
	}

	ensurePollMessageSecret(context.Background(), *origMsg)
	if waMsg := parseE2EMessage(origMsg.Raw); waMsg != nil {
		if sec := waMsg.GetMessageContextInfo().GetMessageSecret(); len(sec) > 0 {
			_ = waClient.Store.MsgSecrets.PutMessageSecret(context.Background(), evt.Info.Chat.ToNonAD(), evt.Info.Sender.ToNonAD(), pollID, sec)
			if creationKey.GetParticipant() != "" {
				if pJID, err := types.ParseJID(creationKey.GetParticipant()); err == nil {
					_ = waClient.Store.MsgSecrets.PutMessageSecret(context.Background(), evt.Info.Chat.ToNonAD(), pJID.ToNonAD(), pollID, sec)
				}
			}
		}
	}

	voteMsg, err := waClient.DecryptPollVote(context.Background(), evt)
	if err != nil {
		log.Printf("decrypting poll vote error: %v", err)
		return
	}

	_, options, _ := extractPollDetails(origMsg.Raw)
	var selected []string
	for _, opt := range options {
		h := sha256.Sum256([]byte(opt))
		for _, selHash := range voteMsg.GetSelectedOptions() {
			if bytes.Equal(h[:], selHash) {
				selected = append(selected, opt)
				break
			}
		}
	}

	_ = msgStore.setPollVotesForVoter(context.Background(), origMsg.ChatJID, pollID, voterJID, selected)
	if evt.Info.IsFromMe {
		_ = msgStore.setPollVotesForVoter(context.Background(), origMsg.ChatJID, pollID, "me", selected)
	}
	if chatJID != origMsg.ChatJID {
		_ = msgStore.setPollVotesForVoter(context.Background(), chatJID, pollID, voterJID, selected)
	}

	glib.IdleAdd(func() {
		if activeMainWindow != nil && activeMainWindow.currentChat != nil {
			if activeMainWindow.currentChat.JID == chatJID || activeMainWindow.currentChat.JID == origMsg.ChatJID {
				activeMainWindow.reloadPollBubble(pollID)
			}
		}
	})
}

// handleIncomingReaction records an incoming reaction from WhatsApp and updates UI.
func handleIncomingReaction(evt *events.Message, rm *waE2E.ReactionMessage) {
	if msgStore == nil || rm == nil || rm.GetKey() == nil {
		return
	}
	targetID := rm.GetKey().GetID()
	if targetID == "" {
		return
	}
	chatJID := evt.Info.Chat.ToNonAD().String()
	if msgStore.isSelfJID(context.Background(), chatJID) {
		if canonical := msgStore.getCanonicalSelfJID(context.Background()); canonical != "" {
			chatJID = canonical
		}
	}
	senderJID := evt.Info.Sender.ToNonAD().String()
	if senderJID == "" {
		senderJID = chatJID
	}
	emoji := rm.GetText()
	senderName := evt.Info.PushName

	if emoji == "" {
		_ = msgStore.removeReaction(context.Background(), chatJID, targetID, senderJID)
		if evt.Info.IsFromMe {
			_ = msgStore.removeMyReaction(context.Background(), chatJID, targetID)
		}
	} else {
		_ = msgStore.saveReaction(context.Background(), Reaction{
			ChatJID:    chatJID,
			MessageID:  targetID,
			SenderJID:  senderJID,
			SenderName: senderName,
			Emoji:      emoji,
			FromMe:     evt.Info.IsFromMe,
			Timestamp:  evt.Info.Timestamp,
		})
	}

	glib.IdleAdd(func() {
		if activeMainWindow != nil {
			activeMainWindow.updateMessageReactions(chatJID, targetID)
		}
	})
}
