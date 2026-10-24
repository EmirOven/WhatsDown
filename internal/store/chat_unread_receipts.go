package store

import (
	"context"
	"time"
)

// ReadReceiptMessages captures receipt targets without loading media payloads.
// The cutoff keeps messages arriving after a bulk-read action unread.
func (s *MessageStore) ReadReceiptMessages(ctx context.Context, chat string, at time.Time, count int) ([]StoredMessage, error) {
	if count <= 0 {
		return nil, nil
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id,sender_jid,timestamp FROM whatsdown_messages
 WHERE chat_jid=? AND from_me=0 AND timestamp<=? ORDER BY timestamp DESC,rowid DESC LIMIT ?`, chat, at.Unix(), count)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var messages []StoredMessage
	for rows.Next() {
		m := StoredMessage{ChatJID: chat}
		var ts int64
		if err := rows.Scan(&m.ID, &m.SenderJID, &ts); err != nil {
			return nil, err
		}
		m.Timestamp = time.Unix(ts, 0)
		messages = append(messages, m)
	}
	return messages, rows.Err()
}
