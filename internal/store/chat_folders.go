package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"go.mau.fi/whatsmeow/types"
)

// ChatFolder is local organization, independent of WhatsApp communities and pins.
// Members are chat JIDs, so folders cannot contain other folders.
type ChatFolder struct {
	ID, Name, Color string
	Collapsed       bool
	ChatJIDs        []string
}

var folderColor = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

func validateFolder(name, color string) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("enter a folder name")
	}
	if !folderColor.MatchString(color) {
		return errors.New("choose a colour in #RRGGBB format")
	}
	return nil
}

func (s *MessageStore) ListChatFolders(ctx context.Context) ([]ChatFolder, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, color, collapsed FROM whatsdown_chat_folders ORDER BY position, id`)
	if err != nil {
		return nil, err
	}
	var folders []ChatFolder
	indices := make(map[string]int)
	for rows.Next() {
		var f ChatFolder
		if err := rows.Scan(&f.ID, &f.Name, &f.Color, &f.Collapsed); err != nil {
			rows.Close()
			return nil, err
		}
		indices[f.ID] = len(folders)
		folders = append(folders, f)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	rows, err = s.db.QueryContext(ctx, `SELECT folder_id, chat_jid FROM whatsdown_chat_folder_members ORDER BY position, chat_jid`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id, jid string
		if err := rows.Scan(&id, &jid); err != nil {
			return nil, err
		}
		if i, ok := indices[id]; ok {
			folders[i].ChatJIDs = append(folders[i].ChatJIDs, jid)
		}
	}
	return folders, rows.Err()
}

func (s *MessageStore) CreateChatFolder(ctx context.Context, name, color string, jids []string) (ChatFolder, error) {
	f := ChatFolder{ID: uuid.NewString(), Name: strings.TrimSpace(name), Color: color}
	if err := validateFolder(name, color); err != nil {
		return f, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return f, err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `INSERT INTO whatsdown_chat_folders(id, name, color, position) VALUES (?, ?, ?, (SELECT COALESCE(MAX(position), -1)+1 FROM whatsdown_chat_folders))`, f.ID, f.Name, color); err != nil {
		return f, err
	}
	if err = moveFolderChats(ctx, tx, jids, f.ID, ""); err != nil {
		return f, err
	}
	return f, tx.Commit()
}

func (s *MessageStore) UpdateChatFolder(ctx context.Context, f ChatFolder) error {
	if err := validateFolder(f.Name, f.Color); err != nil {
		return err
	}
	result, err := s.db.ExecContext(ctx, `UPDATE whatsdown_chat_folders SET name=?, color=?, collapsed=? WHERE id=?`, strings.TrimSpace(f.Name), f.Color, f.Collapsed, f.ID)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err == nil && n == 0 {
		return sql.ErrNoRows
	}
	return err
}

func (s *MessageStore) DeleteChatFolder(ctx context.Context, id string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `DELETE FROM whatsdown_chat_folder_members WHERE folder_id=?`, id); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM whatsdown_chat_folders WHERE id=?`, id); err != nil {
		return err
	}
	return tx.Commit()
}

// MoveFolderChats moves or reorders a block before beforeJID. An empty folder
// removes membership; an empty beforeJID appends to the destination folder.
func (s *MessageStore) MoveFolderChats(ctx context.Context, jids []string, folderID, beforeJID string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := moveFolderChats(ctx, tx, jids, folderID, beforeJID); err != nil {
		return err
	}
	return tx.Commit()
}

func moveFolderChats(ctx context.Context, tx *sql.Tx, jids []string, folderID, before string) error {
	selected := make(map[string]bool)
	var moving []string
	for _, jid := range jids {
		parsed, err := types.ParseJID(jid)
		if err != nil || parsed.User == "" || parsed.Server == "" {
			return fmt.Errorf("invalid chat JID: %s", jid)
		}
		if !selected[jid] {
			selected[jid] = true
			moving = append(moving, jid)
		}
	}
	if len(moving) == 0 {
		return nil
	}
	if folderID != "" {
		var exists int
		if err := tx.QueryRowContext(ctx, `SELECT 1 FROM whatsdown_chat_folders WHERE id=?`, folderID).Scan(&exists); err != nil {
			return err
		}
	}
	if before != "" && selected[before] {
		return nil
	}
	var remaining []string
	if folderID != "" {
		rows, err := tx.QueryContext(ctx, `SELECT chat_jid FROM whatsdown_chat_folder_members WHERE folder_id=? ORDER BY position, chat_jid`, folderID)
		if err != nil {
			return err
		}
		for rows.Next() {
			var jid string
			if err := rows.Scan(&jid); err != nil {
				rows.Close()
				return err
			}
			if !selected[jid] {
				remaining = append(remaining, jid)
			}
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return err
		}
	}
	insert := len(remaining)
	if before != "" {
		insert = -1
		for i, jid := range remaining {
			if jid == before {
				insert = i
				break
			}
		}
		if insert < 0 {
			return errors.New("destination chat is not in this folder")
		}
	}
	for _, jid := range moving {
		if _, err := tx.ExecContext(ctx, `DELETE FROM whatsdown_chat_folder_members WHERE chat_jid=?`, jid); err != nil {
			return err
		}
	}
	if folderID == "" {
		return nil
	}
	ordered := append(append(append([]string{}, remaining[:insert]...), moving...), remaining[insert:]...)
	for i, jid := range ordered {
		if _, err := tx.ExecContext(ctx, `INSERT INTO whatsdown_chat_folder_members(chat_jid, folder_id, position) VALUES (?, ?, ?) ON CONFLICT(chat_jid) DO UPDATE SET folder_id=excluded.folder_id, position=excluded.position`, jid, folderID, i); err != nil {
			return err
		}
	}
	return nil
}

// MoveChatFolder changes only the top-level folder order, never its contents.
func (s *MessageStore) MoveChatFolder(ctx context.Context, id, before string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, `SELECT id FROM whatsdown_chat_folders ORDER BY position, id`)
	if err != nil {
		return err
	}
	var ids []string
	found := false
	for rows.Next() {
		var f string
		if err := rows.Scan(&f); err != nil {
			rows.Close()
			return err
		}
		if f == id {
			found = true
		} else {
			ids = append(ids, f)
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	if !found {
		return sql.ErrNoRows
	}
	if id == before {
		return nil
	}
	insert := len(ids)
	if before != "" {
		insert = -1
		for i, f := range ids {
			if f == before {
				insert = i
				break
			}
		}
		if insert < 0 {
			return sql.ErrNoRows
		}
	}
	ids = append(ids, "")
	copy(ids[insert+1:], ids[insert:])
	ids[insert] = id
	for i, f := range ids {
		if _, err := tx.ExecContext(ctx, `UPDATE whatsdown_chat_folders SET position=? WHERE id=?`, i, f); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *MessageStore) SetChatFolderCollapsed(ctx context.Context, id string, collapsed bool) error {
	_, err := s.db.ExecContext(ctx, `UPDATE whatsdown_chat_folders SET collapsed=? WHERE id=?`, collapsed, id)
	return err
}
