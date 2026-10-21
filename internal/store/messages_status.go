package store

import (
	"context"
	"strings"
)

func (s *MessageStore) UpdateMessageStatuses(ctx context.Context, ids []string, status int) error {
	if len(ids) == 0 {
		return nil
	}
	placeholders := make([]string, len(ids))
	args := make([]any, len(ids)+1)
	args[0] = status
	for i, id := range ids {
		placeholders[i] = "?"
		args[i+1] = id
	}

	query := `
		UPDATE whatsdown_messages
		SET status = MAX(status, ?)
		WHERE id IN (` + strings.Join(placeholders, ",") + `) AND from_me = 1`

	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}
