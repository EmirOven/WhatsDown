package store

import (
	"context"
	"time"
)

func (s *MessageStore) SaveSticker(ctx context.Context, sticker SavedSticker) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO whatsdown_saved_stickers (id, file_path, file_hash, width, height, saved_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			file_path = excluded.file_path,
			file_hash = excluded.file_hash,
			width     = excluded.width,
			height    = excluded.height,
			saved_at  = excluded.saved_at`,
		sticker.ID, sticker.FilePath, sticker.FileHash, sticker.Width, sticker.Height, sticker.SavedAt.Unix())
	return err
}

func (s *MessageStore) DeleteSavedSticker(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM whatsdown_saved_stickers WHERE id = ?`, id)
	return err
}

func (s *MessageStore) ListSavedStickers(ctx context.Context) ([]SavedSticker, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, file_path, file_hash, width, height, saved_at
		FROM whatsdown_saved_stickers
		ORDER BY saved_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stickers []SavedSticker
	for rows.Next() {
		var st SavedSticker
		var savedAtUnix int64
		if err := rows.Scan(&st.ID, &st.FilePath, &st.FileHash, &st.Width, &st.Height, &savedAtUnix); err != nil {
			return nil, err
		}
		st.SavedAt = time.Unix(savedAtUnix, 0)
		stickers = append(stickers, st)
	}
	return stickers, rows.Err()
}

func (s *MessageStore) IsStickerSaved(ctx context.Context, idOrHash string) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM whatsdown_saved_stickers WHERE id = ? OR file_hash = ?`, idOrHash, idOrHash).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
