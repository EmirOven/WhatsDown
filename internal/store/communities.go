package store

import (
	"context"

	"go.mau.fi/whatsmeow/types"
)

func (s *MessageStore) SetChatCommunity(ctx context.Context, jid, parentJID string, isParent, isDefaultSubGroup bool) error {
	pInt := 0
	if isParent {
		pInt = 1
	}
	dInt := 0
	if isDefaultSubGroup {
		dInt = 1
	}
	_, err := s.db.ExecContext(ctx, `
		UPDATE whatsdown_chats
		SET parent_jid = ?, is_parent = ?, is_default_sub_group = ?
		WHERE jid = ?
	`, parentJID, pInt, dInt, jid)
	return err
}

func (s *MessageStore) SyncGroups(ctx context.Context, groups []*types.GroupInfo) error {
	if len(groups) == 0 {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		UPDATE whatsdown_chats
		SET parent_jid = ?, is_parent = ?, is_default_sub_group = ?
		WHERE jid = ?
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, g := range groups {
		if g == nil {
			continue
		}
		parentJID := ""
		if !g.LinkedParentJID.IsEmpty() {
			parentJID = g.LinkedParentJID.String()
		}
		pInt := 0
		if g.IsParent {
			pInt = 1
		}
		dInt := 0
		if g.IsDefaultSubGroup {
			dInt = 1
		}
		_, _ = stmt.ExecContext(ctx, parentJID, pInt, dInt, g.JID.String())
	}

	return tx.Commit()
}

func (s *MessageStore) UpsertChat(ctx context.Context, jid, name string, lastMsgUnix int64) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO whatsdown_chats (jid, name, last_message_at, unread_count)
		VALUES (?, ?, ?, 0)
		ON CONFLICT(jid) DO UPDATE SET
			name = CASE WHEN excluded.name <> '' THEN excluded.name ELSE whatsdown_chats.name END,
			last_message_at = MAX(whatsdown_chats.last_message_at, excluded.last_message_at)
	`, jid, name, lastMsgUnix)
	return err
}
