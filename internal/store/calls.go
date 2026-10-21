package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

func FormatCallDuration(secs int) string {
	if secs <= 0 {
		return ""
	}
	m := secs / 60
	s := secs % 60
	if m == 0 {
		return fmt.Sprintf("%ds", s)
	}
	return fmt.Sprintf("%dm %ds", m, s)
}

func (s *MessageStore) SaveCallTx(ctx context.Context, tx execer, call CallRecord) error {
	fromMeInt, vidInt, grpInt := 0, 0, 0
	if call.FromMe {
		fromMeInt = 1
	}
	if call.IsVideo {
		vidInt = 1
	}
	if call.IsGroup {
		grpInt = 1
	}
	query := `INSERT OR IGNORE INTO whatsdown_calls (id, chat_jid, caller_jid, caller_name, timestamp, duration_secs, is_video, is_group, outcome, from_me) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := tx.ExecContext(ctx, query, call.ID, call.ChatJID, call.CallerJID, call.CallerName, call.Timestamp.Unix(), call.DurationSecs, vidInt, grpInt, call.Outcome, fromMeInt)
	return err
}

func (s *MessageStore) SaveCall(ctx context.Context, call CallRecord) error {
	return s.SaveCallTx(ctx, s.db, call)
}

func (s *MessageStore) ListCalls(ctx context.Context, query string) ([]CallRecord, error) {
	sqlQuery := `
		SELECT id, chat_jid, caller_jid, caller_name, timestamp, duration_secs, is_video, is_group, outcome, from_me
		FROM whatsdown_calls
		ORDER BY timestamp DESC LIMIT 200`
	var rows *sql.Rows
	var err error
	if query != "" {
		sqlQuery = `
			SELECT id, chat_jid, caller_jid, caller_name, timestamp, duration_secs, is_video, is_group, outcome, from_me
			FROM whatsdown_calls
			WHERE caller_name LIKE ? OR caller_jid LIKE ?
			ORDER BY timestamp DESC LIMIT 200`
		q := "%" + query + "%"
		rows, err = s.db.QueryContext(ctx, sqlQuery, q, q)
	} else {
		rows, err = s.db.QueryContext(ctx, sqlQuery)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []CallRecord
	for rows.Next() {
		var (
			c       CallRecord
			ts      int64
			vidInt  int
			grpInt  int
			fromInt int
		)
		err := rows.Scan(&c.ID, &c.ChatJID, &c.CallerJID, &c.CallerName, &ts, &c.DurationSecs, &vidInt, &grpInt, &c.Outcome, &fromInt)
		if err != nil {
			return nil, err
		}
		c.Timestamp = time.Unix(ts, 0)
		c.IsVideo = vidInt != 0
		c.IsGroup = grpInt != 0
		c.FromMe = fromInt != 0
		records = append(records, c)
	}
	return records, rows.Err()
}

func (s *MessageStore) DeleteCall(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM whatsdown_calls WHERE id = ?`, id)
	return err
}

func (s *MessageStore) ClearCalls(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM whatsdown_calls`)
	return err
}
