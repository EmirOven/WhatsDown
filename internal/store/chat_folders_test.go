package store_test

import (
	"reflect"
	"testing"

	"postbetasoftware.com/whatsdown/internal/store"
)

func TestChatFolderPersistenceAndMoves(t *testing.T) {
	s, ctx, db := setupTestStore(t)
	defer db.Close()
	a, b, c := "1@s.whatsapp.net", "2@s.whatsapp.net", "3@g.us"
	first, err := s.CreateChatFolder(ctx, " Friends ", "#3584e4", []string{a, b, a})
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.CreateChatFolder(ctx, "Work", "#e01b24", []string{c})
	if err != nil {
		t.Fatal(err)
	}
	check := func(want ...[]string) []store.ChatFolder {
		t.Helper()
		folders, err := s.ListChatFolders(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if len(folders) != len(want) {
			t.Fatalf("folders = %d, want %d", len(folders), len(want))
		}
		for i := range want {
			if !reflect.DeepEqual(folders[i].ChatJIDs, want[i]) {
				t.Fatalf("folder %d members = %v, want %v", i, folders[i].ChatJIDs, want[i])
			}
		}
		return folders
	}
	folders := check([]string{a, b}, []string{c})
	if folders[0].Name != "Friends" {
		t.Fatal("folder name not trimmed")
	}
	if err := s.MoveFolderChats(ctx, []string{b}, first.ID, a); err != nil {
		t.Fatal(err)
	}
	check([]string{b, a}, []string{c})
	if err := s.MoveFolderChats(ctx, []string{a, b}, second.ID, c); err != nil {
		t.Fatal(err)
	}
	check(nil, []string{a, b, c})
	if err := s.MoveFolderChats(ctx, []string{b}, "", ""); err != nil {
		t.Fatal(err)
	}
	check(nil, []string{a, c})
	if err := s.MoveChatFolder(ctx, second.ID, first.ID); err != nil {
		t.Fatal(err)
	}
	folders = check([]string{a, c}, nil)
	if folders[0].ID != second.ID {
		t.Fatal("folder reorder did not persist")
	}
	folders[0].Name, folders[0].Color, folders[0].Collapsed = "Office", "#00ff88", true
	if err := s.UpdateChatFolder(ctx, folders[0]); err != nil {
		t.Fatal(err)
	}
	// Reopening the store must not reset folder metadata or ordering.
	reopened, err := store.OpenMessageStore(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	saved, err := reopened.ListChatFolders(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if saved[0].Name != "Office" || saved[0].Color != "#00ff88" || !saved[0].Collapsed {
		t.Fatalf("metadata lost: %+v", saved[0])
	}
	if err := s.DeleteChatFolder(ctx, second.ID); err != nil {
		t.Fatal(err)
	}
	check(nil)
	// Removing a folder releases its old chats for organization elsewhere.
	if err := s.MoveFolderChats(ctx, []string{a}, first.ID, ""); err != nil {
		t.Fatal(err)
	}
	check([]string{a})
}

func TestChatFolderInvalidMovesAreAtomic(t *testing.T) {
	s, ctx, db := setupTestStore(t)
	defer db.Close()
	a := "1@s.whatsapp.net"
	first, err := s.CreateChatFolder(ctx, "Friends", "#3584e4", []string{a})
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.CreateChatFolder(ctx, "Work", "#ff0000", nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, move := range []struct {
		jids           []string
		folder, before string
	}{
		{[]string{a}, "missing", ""},
		{[]string{first.ID}, second.ID, ""}, // A folder cannot be a member.
		{[]string{a}, second.ID, "not-a-member@s.whatsapp.net"},
	} {
		if err := s.MoveFolderChats(ctx, move.jids, move.folder, move.before); err == nil {
			t.Fatalf("invalid move accepted: %+v", move)
		}
		folders, err := s.ListChatFolders(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(folders[0].ChatJIDs, []string{a}) || len(folders[1].ChatJIDs) != 0 {
			t.Fatal("invalid move changed memberships")
		}
	}
	for _, value := range []struct{ name, color string }{{" ", "#3584e4"}, {"Friends", "red; invalid CSS"}} {
		if _, err := s.CreateChatFolder(ctx, value.name, value.color, []string{a}); err == nil {
			t.Fatal("invalid metadata accepted")
		}
	}
	if err := s.MoveChatFolder(ctx, first.ID, "missing"); err == nil {
		t.Fatal("invalid folder order accepted")
	}
	folders, _ := s.ListChatFolders(ctx)
	if folders[0].ID != first.ID {
		t.Fatal("invalid reorder changed folder order")
	}
}
