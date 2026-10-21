package store

import (
	"context"

	"go.mau.fi/whatsmeow/types"
)

func (s *MessageStore) SetChatPinned(ctx context.Context, jid string, pinned bool) error {
	v := 0
	if pinned {
		v = 1
	}
	_, err := s.db.ExecContext(ctx, `UPDATE whatsdown_chats SET pinned = ? WHERE jid = ?`, v, jid)
	return err
}

func (s *MessageStore) SetChatArchived(ctx context.Context, jid string, archived bool) error {
	v := 0
	if archived {
		v = 1
	}
	_, err := s.db.ExecContext(ctx, `UPDATE whatsdown_chats SET archived = ? WHERE jid = ?`, v, jid)
	return err
}

func (s *MessageStore) SetChatMuted(ctx context.Context, jid string, muted bool) error {
	v := 0
	if muted {
		v = 1
	}
	_, err := s.db.ExecContext(ctx, `UPDATE whatsdown_chats SET muted = ? WHERE jid = ?`, v, jid)
	return err
}

func (s *MessageStore) MarkChatRead(ctx context.Context, chat types.JID) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE whatsdown_chats SET unread_count = 0 WHERE jid = ?`,
		chat.String(),
	)
	return err
}
