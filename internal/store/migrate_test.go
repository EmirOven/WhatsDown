package store_test

import (
	"reflect"
	"testing"

	"postbetasoftware.com/whatsdown/internal/store"
)

func TestFolderMigrationPreservesExistingTables(t *testing.T) {
	for _, partial := range []bool{false, true} {
		name := "existing schema"
		if partial {
			name = "partial schema"
		}
		t.Run(name, func(t *testing.T) {
			s, ctx, db := setupTestStore(t)
			defer db.Close()
			folder, err := s.CreateChatFolder(ctx, "Friends", "#3584e4", []string{"1@s.whatsapp.net"})
			if err != nil {
				t.Fatal(err)
			}
			if partial {
				if _, err := db.ExecContext(ctx, `DROP TABLE whatsdown_chat_folder_members`); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := db.ExecContext(ctx, `UPDATE whatsdown_version SET version = 15`); err != nil {
				t.Fatal(err)
			}
			reopened, err := store.OpenMessageStore(ctx, db)
			if err != nil {
				t.Fatalf("reopening existing folder schema: %v", err)
			}
			folders, err := reopened.ListChatFolders(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if len(folders) != 1 || folders[0].ID != folder.ID || folders[0].Name != "Friends" {
				t.Fatalf("folder data changed: %+v", folders)
			}
			if !partial && !reflect.DeepEqual(folders[0].ChatJIDs, []string{"1@s.whatsapp.net"}) {
				t.Fatalf("folder members changed: %v", folders[0].ChatJIDs)
			}
			var version int
			if err := db.QueryRowContext(ctx, `SELECT version FROM whatsdown_version`).Scan(&version); err != nil {
				t.Fatal(err)
			}
			if version != 18 {
				t.Fatalf("schema version = %d, want 18", version)
			}
		})
	}
}
