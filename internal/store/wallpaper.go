package store

import (
	"context"
	"database/sql"
	"time"
)

// ChatWallpaper represents the custom background setting for a conversation.
type ChatWallpaper struct {
	ChatJID string
	Kind    string // "none", "color", "image"
	Value   string // hex/rgba string or file path
}

// GetChatWallpaper returns the wallpaper configuration for a chat.
func (s *MessageStore) GetChatWallpaper(ctx context.Context, chatJID string) (ChatWallpaper, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT chat_jid, kind, value
		FROM whatsdown_chat_wallpapers
		WHERE chat_jid = ?
	`, chatJID)

	var w ChatWallpaper
	err := row.Scan(&w.ChatJID, &w.Kind, &w.Value)
	if err == sql.ErrNoRows {
		return ChatWallpaper{ChatJID: chatJID, Kind: "none"}, nil
	}
	if err != nil {
		return ChatWallpaper{ChatJID: chatJID, Kind: "none"}, err
	}
	return w, nil
}

// SetChatWallpaper updates or inserts the wallpaper setting for a chat.
func (s *MessageStore) SetChatWallpaper(ctx context.Context, chatJID, kind, value string) error {
	now := time.Now().Unix()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO whatsdown_chat_wallpapers (chat_jid, kind, value, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(chat_jid) DO UPDATE SET
			kind = excluded.kind,
			value = excluded.value,
			updated_at = excluded.updated_at
	`, chatJID, kind, value, now)
	return err
}
