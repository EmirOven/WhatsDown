package ui

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

func TestContextMenuAddsSelectedChatsToFolder(t *testing.T) {
	c := dialogTestController(t)
	ctx := context.Background()
	a, b := "0@a.test", "0@b.test"
	if _, err := c.store.CreateChatFolder(ctx, "Existing", "#3584e4", []string{a}); err != nil {
		t.Fatal(err)
	}
	destination, err := c.store.CreateChatFolder(ctx, "Friends", "#00aa00", nil)
	if err != nil {
		t.Fatal(err)
	}
	c.LoadChats()
	c.selectedChatJIDs = map[string]bool{a: true, b: true}
	var add *gtk.Button
	for i, item := range c.displayedChatRows {
		if item.chat.JID != a {
			continue
		}
		walkDialog(c.chatsList.RowAtIndex(i), func(w gtk.Widgetter) {
			if button, ok := w.(*gtk.Button); ok {
				walkDialog(button, func(child gtk.Widgetter) {
					if label, ok := child.(*gtk.Label); ok && label.Text() == "Add to Folder…" {
						add = button
					}
				})
			}
		})
	}
	if add == nil {
		t.Fatal("chat context menu has no Add to Folder action")
	}
	c.window.Present()
	add.Emit("clicked")
	dialog := c.window.VisibleDialog()
	if dialog == nil {
		t.Fatal("Add to Folder did not open a native picker")
	}
	var target *adw.ActionRow
	walkDialog(dialog, func(w gtk.Widgetter) {
		if row, ok := w.(*adw.ActionRow); ok && row.Title() == "Friends" {
			target = row
		}
	})
	if target == nil || !target.Activatable() {
		t.Fatal("destination folder missing")
	}
	target.Emit("activated")
	pumpSidebarEvents(40 * time.Millisecond)
	folders, err := c.store.ListChatFolders(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, folder := range folders {
		if folder.ID == destination.ID {
			if !folderContainsChats(folder, []string{a, b}) || len(folder.ChatJIDs) != 2 {
				t.Fatalf("selected chats were not moved: %+v", folder)
			}
		} else if len(folder.ChatJIDs) != 0 {
			t.Fatalf("old membership retained: %+v", folder)
		}
	}
}

func TestReadAllButtonUsesThemedIconAndShortLabel(t *testing.T) {
	_ = newFolderSidebar(t)
	xml, err := os.ReadFile("../../ui/components/ChatsMenu.ui")
	if err != nil {
		t.Fatal(err)
	}
	builder := gtk.NewBuilder()
	if err := builder.AddFromString(string(xml)); err != nil {
		t.Fatal(err)
	}
	button, ok := castObject[*gtk.Button](builder, "mark_all_read_btn")
	if !ok {
		t.Fatal("Read All button missing")
	}
	content, ok := button.Child().(*adw.ButtonContent)
	if !ok || content.Label() != "Read All" || content.IconName() != "mail-read-symbolic" {
		t.Fatal("Read All does not use native button content")
	}
	if !gtk.IconThemeGetForDisplay(gdk.DisplayGetDefault()).HasIcon(content.IconName()) {
		t.Fatal("Read All icon unavailable")
	}
}
