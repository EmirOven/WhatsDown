package ui

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
	"time"

	coreglib "github.com/diamondburned/gotk4/pkg/core/glib"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	_ "github.com/mattn/go-sqlite3"
	"postbetasoftware.com/whatsdown/internal/store"
)

func newFolderSidebar(t *testing.T) *MainController {
	t.Helper()
	if os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
		t.Skip("no display available")
	}
	runtime.LockOSThread()
	t.Cleanup(runtime.UnlockOSThread)
	gtk.Init()
	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "sidebar.db")+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	s, err := store.OpenMessageStore(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	c := &MainController{store: s, expandedCommunities: make(map[string]bool)}
	c.chatsList, c.archivedChatsList = gtk.NewListBox(), gtk.NewListBox()
	c.bindChatListSelection(c.chatsList, false)
	c.bindChatListSelection(c.archivedChatsList, true)
	t.Cleanup(func() {
		for _, cleanup := range c.chatMenuCleanups {
			cleanup()
		}
		c.releaseMessageWidget(c.chatsList)
		c.releaseMessageWidget(c.archivedChatsList)
		c.chatsList.RemoveAll()
		c.archivedChatsList.RemoveAll()
	})
	for i, jid := range []string{"0@a.test", "0@b.test", "0@c.test", "0@d.test"} {
		if err := s.UpsertChat(context.Background(), jid, jid, int64(100-i)); err != nil {
			t.Fatal(err)
		}
	}
	return c
}

func TestFolderLayoutSelectionAndSearch(t *testing.T) {
	c := newFolderSidebar(t)
	ctx := context.Background()
	f, err := c.store.CreateChatFolder(ctx, "Friends", "#3584e4", []string{"0@c.test", "0@a.test"})
	if err != nil {
		t.Fatal(err)
	}
	c.LoadChats()
	rows := c.displayedChatRows
	if len(rows) != 6 || !rows[0].isFolderHeader || rows[1].chat.JID != "0@c.test" || rows[2].chat.JID != "0@a.test" || !rows[3].isLooseDropZone || rows[4].chat.JID != "0@b.test" {
		t.Fatalf("unexpected folder/loose layout: %+v", rows)
	}
	c.chatsList.SelectRow(c.chatsList.RowAtIndex(1))
	c.chatsList.SelectRow(c.chatsList.RowAtIndex(4))
	if len(c.selectedChatJIDs) != 2 {
		t.Fatalf("selection = %v", c.selectedChatJIDs)
	}
	c.LoadChats()
	if len(c.chatsList.SelectedRows()) != 2 {
		t.Fatal("refresh lost multi-selection")
	}
	c.selectChatForContext(c.chatsList.RowAtIndex(1), rows[1].chat, false)
	if got := c.selectedFolderChats(rows[1].chat); !reflect.DeepEqual(got, []string{"0@c.test", "0@b.test"}) {
		t.Fatalf("context selection = %v", got)
	}
	c.selectChatForContext(c.chatsList.RowAtIndex(2), rows[2].chat, false)
	if len(c.selectedChatJIDs) != 1 || !c.selectedChatJIDs["0@a.test"] {
		t.Fatal("right-click did not select an unselected chat")
	}
	c.toggleChatFolder(f.ID)
	visible := 0
	for i := range c.displayedChatRows {
		if c.chatsList.RowAtIndex(i).Visible() {
			visible++
		}
	}
	if visible != 4 {
		t.Fatal("collapse did not hide folder members")
	}
	pumpSidebarEvents(30 * time.Millisecond)
	c.chatsSearchFilter = "0@c.test"
	c.LoadChats()
	if len(c.displayedChatRows) != 3 || c.displayedChatRows[1].chat.JID != "0@c.test" {
		t.Fatalf("search did not reveal collapsed folder match: %+v", c.displayedChatRows)
	}
	saved, _ := c.store.ListChatFolders(ctx)
	if !saved[0].Collapsed {
		t.Fatal("search overwrote collapse preference")
	}
}

func TestFolderDropOrderingAndUngrouping(t *testing.T) {
	c := newFolderSidebar(t)
	ctx := context.Background()
	f, err := c.store.CreateChatFolder(ctx, "Friends", "#3584e4", []string{"0@b.test", "0@c.test"})
	if err != nil {
		t.Fatal(err)
	}
	g, err := c.store.CreateChatFolder(ctx, "Work", "#ff0000", nil)
	if err != nil {
		t.Fatal(err)
	}
	c.LoadChats()
	if err := c.applyChatDrop(chatDragPayload{ChatJIDs: []string{"0@a.test", "0@d.test"}}, chatRowItem{folderID: f.ID, chat: store.Chat{JID: "0@c.test"}}, false); err != nil {
		t.Fatal(err)
	}
	c.LoadChats()
	if got := c.chatFolders[0].ChatJIDs; !reflect.DeepEqual(got, []string{"0@b.test", "0@a.test", "0@d.test", "0@c.test"}) {
		t.Fatalf("drop order = %v", got)
	}
	if err := c.applyChatDrop(chatDragPayload{ChatJIDs: []string{"0@d.test", "0@a.test"}}, chatRowItem{isLooseDropZone: true}, false); err != nil {
		t.Fatal(err)
	}
	c.LoadChats()
	var loose []string
	for _, row := range c.displayedChatRows {
		if !row.isFolderHeader && !row.isLooseDropZone && row.folderID == "" {
			loose = append(loose, row.chat.JID)
		}
	}
	if !reflect.DeepEqual(loose, []string{"0@a.test", "0@d.test"}) {
		t.Fatalf("ungrouped chats did not resume date order: %v", loose)
	}
	if err := c.applyChatDrop(chatDragPayload{FolderID: g.ID}, chatRowItem{isFolderHeader: true, folderID: f.ID}, false); err != nil {
		t.Fatal(err)
	}
	c.LoadChats()
	if c.chatFolders[0].ID != g.ID || len(c.chatFolders[0].ChatJIDs) != 0 {
		t.Fatal("folder drop nested folders instead of reordering them")
	}
	// Arrival times must not disturb manual folder order. WhatsApp pins
	// move loose chats above unpinned chats.
	if err := c.store.UpsertChat(ctx, "0@c.test", "C", 999); err != nil {
		t.Fatal(err)
	}
	if err := c.store.SetChatPinned(ctx, "0@d.test", true); err != nil {
		t.Fatal(err)
	}
	c.LoadChats()
	if !reflect.DeepEqual(c.chatFolders[1].ChatJIDs, []string{"0@b.test", "0@c.test"}) {
		t.Fatal("new messages changed manual member order")
	}
	loose = nil
	for _, row := range c.displayedChatRows {
		if row.folderID == "" && !row.isLooseDropZone {
			loose = append(loose, row.chat.JID)
		}
	}
	if !reflect.DeepEqual(loose, []string{"0@d.test", "0@a.test"}) {
		t.Fatalf("pinned chat is not first: %v", loose)
	}
}

func TestDecodeChatDrag(t *testing.T) {
	for _, invalid := range []string{"file:///tmp/photo.jpg", "https://example.com", chatDragPrefix + `{}`, chatDragPrefix + `{"folder":"a","chats":["1@s.whatsapp.net"]}`} {
		if _, ok := decodeChatDrag(invalid); ok {
			t.Fatalf("accepted invalid drag payload %q", invalid)
		}
	}
	encoded, _ := json.Marshal(chatDragPayload{ChatJIDs: []string{"1@s.whatsapp.net"}})
	if p, ok := decodeChatDrag(chatDragPrefix + string(encoded)); !ok || len(p.ChatJIDs) != 1 {
		t.Fatal("valid drag rejected")
	}
}

func TestLooseChatPinnedOrder(t *testing.T) {
	now := time.Now()
	chats := []store.Chat{{JID: "old", Pinned: true, LastMessageAt: now.Add(-time.Hour)}, {JID: "recent", LastMessageAt: now}}
	top, _ := groupCommunities(chats, "", nil)
	if top[0].JID != "old" {
		t.Fatal("pinned loose chat is not first")
	}
}

func TestFolderDialogAndCleanup(t *testing.T) {
	c := newFolderSidebar(t)
	c.LoadChats()
	baseline := len(c.widgetSignalCleanups)
	for i := 0; i < 3; i++ {
		c.showChatFolderDialog(nil, []string{"0@a.test", "0@b.test"})
		var dialog *gtk.Window
		for _, widget := range gtk.WindowListToplevels() {
			if w, ok := widget.(*gtk.Window); ok && w.Title() == "Create folder" {
				dialog = w
				break
			}
		}
		if dialog == nil {
			t.Fatal("folder dialog not shown")
		}
		if i < 2 {
			dialog.Close()
			if len(c.widgetSignalCleanups) != baseline {
				t.Fatal("cancelled folder dialog retained callbacks")
			}
			continue
		}
		t.Cleanup(func() {
			if dialog.Visible() {
				dialog.Close()
			}
		})
		var name *gtk.Entry
		var color *gtk.ColorDialogButton
		var create *gtk.Button
		var inspect func(gtk.Widgetter)
		inspect = func(widget gtk.Widgetter) {
			switch w := widget.(type) {
			case *gtk.Entry:
				name = w
			case *gtk.ColorDialogButton:
				color = w
			case *gtk.Button:
				if w.Label() == "Create" {
					create = w
				}
			}
			for child := gtk.BaseWidget(widget).FirstChild(); child != nil; child = gtk.BaseWidget(child).NextSibling() {
				inspect(child)
			}
		}
		inspect(dialog)
		if name == nil || color == nil || create == nil {
			t.Fatal("missing folder dialog controls")
		}
		create.Emit("clicked")
		folders, _ := c.store.ListChatFolders(context.Background())
		if len(folders) != 0 {
			t.Fatal("empty folder name accepted")
		}
		name.SetText("Friends")
		rgba := gdk.NewRGBA(1, 0.5, 0, 1)
		color.SetRGBA(&rgba)
		create.Emit("clicked")
		folders, err := c.store.ListChatFolders(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if len(folders) != 1 || folders[0].Name != "Friends" || folders[0].Color != "#ff8000" || len(folders[0].ChatJIDs) != 2 {
			t.Fatalf("folder creation from dialog failed: %+v", folders)
		}
		// Repeated sync/list refreshes retain callbacks for current widgets only.
		withFolder := len(c.widgetSignalCleanups)
		for n := 0; n < 20; n++ {
			c.LoadChats()
		}
		if len(c.widgetSignalCleanups) != withFolder {
			t.Fatal("folder list rebuilds accumulated callbacks")
		}
	}
}

func TestFolderedCommunityDoesNotAppearTwice(t *testing.T) {
	chats := []store.Chat{{JID: "community@g.us", IsParent: true}, {JID: "child@g.us", ParentJID: "community@g.us"}}
	grouped, loose := folderChatSections(chats, []store.ChatFolder{{ID: "folder", ChatJIDs: []string{"community@g.us"}}})
	if len(grouped["folder"]) != 1 {
		t.Fatal("community missing from manual folder")
	}
	top, _ := groupCommunities(loose, "", nil)
	if len(top) != 1 || top[0].JID != "child@g.us" {
		t.Fatal("foldered community was synthesized again outside its folder")
	}
}

func TestFolderChatActivationReplacesLooseSelection(t *testing.T) {
	c := newFolderSidebar(t)
	if _, err := c.store.CreateChatFolder(context.Background(), "Friends", "#3584e4", []string{"0@c.test"}); err != nil {
		t.Fatal(err)
	}
	c.LoadChats()
	var folderIndex, looseIndex int
	for i, item := range c.displayedChatRows {
		if item.chat.JID == "0@c.test" {
			folderIndex = i
		}
		if item.chat.JID == "0@a.test" {
			looseIndex = i
		}
	}
	c.chatsList.SelectRow(c.chatsList.RowAtIndex(looseIndex))
	folderChat := c.displayedChatRows[folderIndex].chat
	c.currentChat = &folderChat
	// Activating an already open chat also synchronizes the sidebar without
	// needing to rebuild the full chat view.
	c.chatsList.RowAtIndex(folderIndex).Activate()
	c.LoadChats()
	selected := c.chatsList.SelectedRows()
	if len(selected) != 1 || c.displayedChatRows[selected[0].Index()].chat.JID != folderChat.JID {
		t.Fatalf("folder activation left wrong selection: %v", c.selectedChatJIDs)
	}
}

func TestFolderToggleKeepsRowsAndScrollPosition(t *testing.T) {
	c := newFolderSidebar(t)
	folder, err := c.store.CreateChatFolder(context.Background(), "Folder", "#3584e4", []string{"0@a.test", "0@b.test"})
	if err != nil {
		t.Fatal(err)
	}
	c.LoadChats()
	header := c.chatsList.RowAtIndex(0)
	child := c.chatsList.RowAtIndex(1)
	scroll := gtk.NewScrolledWindow()
	scroll.SetChild(c.chatsList)
	adjustment := scroll.VAdjustment()
	adjustment.SetUpper(2000)
	adjustment.SetPageSize(200)
	adjustment.SetValue(120)
	c.toggleChatFolder(folder.ID)
	if child.Visible() {
		t.Fatal("collapse left member visible")
	}
	if coreglib.BaseObject(c.chatsList.RowAtIndex(0)).Native() != coreglib.BaseObject(header).Native() {
		t.Fatal("collapse rebuilt the header")
	}
	c.toggleChatFolder(folder.ID)
	if !child.Visible() || coreglib.BaseObject(c.chatsList.RowAtIndex(1)).Native() != coreglib.BaseObject(child).Native() {
		t.Fatal("expansion rebuilt existing chat rows")
	}
	pumpSidebarEvents(30 * time.Millisecond)
	if adjustment.Value() != 120 {
		t.Fatalf("folder toggle moved scroll to %v", adjustment.Value())
	}
	deadline := time.Now().Add(time.Second)
	for {
		folders, err := c.store.ListChatFolders(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if len(folders) == 1 && !folders[0].Collapsed {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("rapid toggles persisted out of order")
		}
		pumpSidebarEvents(10 * time.Millisecond)
	}
}
