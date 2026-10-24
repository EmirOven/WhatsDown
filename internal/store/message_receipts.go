package store

import (
	"context"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/proto/waWeb"
	"go.mau.fi/whatsmeow/types"
)

type MessageReceipt struct {
	RecipientJID string
	Expected     bool
	Status       int
	DeliveredAt  time.Time
	ReadAt       time.Time
}

func (s *MessageStore) receiptJID(ctx context.Context, jid string) string {
	if parsed, err := types.ParseJID(jid); err == nil {
		jid = parsed.ToNonAD().String()
	}
	if strings.HasSuffix(jid, "@lid") {
		if pn := s.ResolveLIDtoPN(ctx, jid); pn != "" {
			return pn
		}
	}
	return jid
}

// SetMessageRecipients freezes the recipient set for this message at send time.
func (s *MessageStore) SetMessageRecipients(ctx context.Context, chat, id string, recipients []string) error {
	normalized := make([]string, 0, len(recipients))
	for _, jid := range recipients {
		if jid != "" && !(strings.HasSuffix(chat, "@g.us") && s.IsSelfJID(ctx, jid)) {
			normalized = append(normalized, s.receiptJID(ctx, jid))
		}
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO whatsdown_message_recipient_sets(chat_jid, message_id) VALUES (?, ?)`, chat, id)
	if err != nil {
		return err
	}
	inserted, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if inserted == 0 {
		return tx.Commit()
	}
	for _, jid := range normalized {
		if _, err := tx.ExecContext(ctx, `INSERT INTO whatsdown_message_recipients(chat_jid, message_id, recipient_jid, expected) VALUES (?, ?, ?, 1) ON CONFLICT(chat_jid, message_id, recipient_jid) DO UPDATE SET expected = 1`, chat, id, jid); err != nil {
			return err
		}
	}
	if err := aggregateReceiptStatus(ctx, tx, chat, id); err != nil {
		return err
	}
	return tx.Commit()
}

func aggregateReceiptStatus(ctx context.Context, tx execer, chat, id string) error {
	if err := mergeReceiptIdentitiesTx(ctx, tx, chat, id); err != nil {
		return err
	}
	if strings.HasSuffix(chat, "@g.us") {
		// The author is not a recipient. Also repair previously recorded self rows.
		if _, err := tx.ExecContext(ctx, `DELETE FROM whatsdown_message_recipients WHERE chat_jid = ? AND message_id = ? AND recipient_jid IN (
   SELECT sender_jid FROM whatsdown_messages WHERE chat_jid = ? AND id = ? AND from_me = 1
   UNION SELECT lm.pn || '@s.whatsapp.net' FROM whatsmeow_lid_map lm JOIN whatsdown_messages m
   ON m.sender_jid = lm.lid || '@lid' WHERE m.chat_jid = ? AND m.id = ? AND m.from_me = 1
  )`, chat, id, chat, id, chat, id); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `UPDATE whatsdown_messages SET status = MAX(status, CASE
   WHEN EXISTS (SELECT 1 FROM whatsdown_message_recipient_sets WHERE chat_jid = ? AND message_id = ?)
   THEN COALESCE((SELECT MIN(status) FROM whatsdown_message_recipients WHERE chat_jid = ? AND message_id = ? AND expected = 1), 1)
   ELSE 1 END) WHERE chat_jid = ? AND id = ? AND from_me = 1`, chat, id, chat, id, chat, id)
		return err
	}
	_, err := tx.ExecContext(ctx, `UPDATE whatsdown_messages SET status = MAX(status, COALESCE((SELECT MAX(status) FROM whatsdown_message_recipients WHERE chat_jid = ? AND message_id = ?), 1)) WHERE chat_jid = ? AND id = ? AND from_me = 1`, chat, id, chat, id)
	return err
}

// SaveMessageReceipt records a recipient's receipt without promoting a whole group.
func (s *MessageStore) SaveMessageReceipt(ctx context.Context, chat string, ids []string, recipient string, status int, at time.Time) error {
	if recipient == "" || status < MessageStatusDelivered || status > MessageStatusRead {
		return nil
	}
	chat = s.receiptJID(ctx, chat)
	recipient = s.receiptJID(ctx, recipient)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, id := range ids {
		if err := saveReceiptTx(ctx, tx, chat, id, recipient, status, at); err != nil {
			return err
		}
		if err := aggregateReceiptStatus(ctx, tx, chat, id); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func saveReceiptTx(ctx context.Context, tx execer, chat, id, recipient string, status int, at time.Time) error {
	delivered, read := int64(0), int64(0)
	if !at.IsZero() {
		if status == MessageStatusDelivered {
			delivered = at.Unix()
		}
		if status == MessageStatusRead {
			read = at.Unix()
		}
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO whatsdown_message_recipients(chat_jid, message_id, recipient_jid, status, delivered_at, read_at) VALUES (?, ?, ?, ?, ?, ?)
 ON CONFLICT(chat_jid, message_id, recipient_jid) DO UPDATE SET
 status = MAX(status, excluded.status),
 delivered_at = CASE WHEN excluded.delivered_at > 0 AND (delivered_at = 0 OR excluded.delivered_at < delivered_at) THEN excluded.delivered_at ELSE delivered_at END,
 read_at = CASE WHEN excluded.read_at > 0 AND (read_at = 0 OR excluded.read_at < read_at) THEN excluded.read_at ELSE read_at END`, chat, id, recipient, status, delivered, read)
	return err
}

func (s *MessageStore) GetMessageReceipts(ctx context.Context, chat, id string) ([]MessageReceipt, bool, error) {
	selfPN, selfLID := s.GetSelfJIDs(ctx)
	// A LID mapping may arrive after its receipt. Reconcile it before displaying.
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, false, err
	}
	defer tx.Rollback()
	if strings.HasSuffix(chat, "@g.us") {
		if _, err := tx.ExecContext(ctx, `DELETE FROM whatsdown_message_recipients WHERE chat_jid = ? AND message_id = ? AND recipient_jid IN (?, ?)`, chat, id, selfPN, selfLID); err != nil {
			return nil, false, err
		}
	}
	if err := aggregateReceiptStatus(ctx, tx, chat, id); err != nil {
		return nil, false, err
	}
	if err := tx.Commit(); err != nil {
		return nil, false, err
	}
	var known bool
	if err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM whatsdown_message_recipient_sets WHERE chat_jid = ? AND message_id = ?)`, chat, id).Scan(&known); err != nil {
		return nil, false, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT recipient_jid, expected, status, delivered_at, read_at FROM whatsdown_message_recipients WHERE chat_jid = ? AND message_id = ? ORDER BY status DESC, recipient_jid`, chat, id)
	if err != nil {
		return nil, known, err
	}
	defer rows.Close()
	var receipts []MessageReceipt
	for rows.Next() {
		var r MessageReceipt
		var delivered, read int64
		if err := rows.Scan(&r.RecipientJID, &r.Expected, &r.Status, &delivered, &read); err != nil {
			return nil, known, err
		}
		if delivered > 0 {
			r.DeliveredAt = time.Unix(delivered, 0)
		}
		if read > 0 {
			r.ReadAt = time.Unix(read, 0)
		}
		receipts = append(receipts, r)
	}
	return receipts, known, rows.Err()
}

// Import only metadata supplied by WhatsApp, including recipients with no receipt
// timestamps yet. A partial UserReceipt list does not prove the full send roster.
func saveSyncedReceiptsTx(ctx context.Context, tx execer, chat, id string, web *waWeb.WebMessageInfo) error {
	for _, r := range web.GetUserReceipt() {
		jid := r.GetUserJID()
		if jid == "" {
			continue
		}
		if parsed, err := types.ParseJID(jid); err == nil {
			jid = parsed.ToNonAD().String()
		}
		if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO whatsdown_message_recipients
   (chat_jid, message_id, recipient_jid) VALUES (?, ?, ?)`, chat, id, jid); err != nil {
			return err
		}
		if ts := r.GetReceiptTimestamp(); ts > 0 {
			if err := saveReceiptTx(ctx, tx, chat, id, jid, MessageStatusDelivered, time.Unix(ts, 0)); err != nil {
				return err
			}
		}
		ts := r.GetReadTimestamp()
		if ts == 0 {
			ts = r.GetPlayedTimestamp()
		}
		if ts > 0 {
			if err := saveReceiptTx(ctx, tx, chat, id, jid, MessageStatusRead, time.Unix(ts, 0)); err != nil {
				return err
			}
		}
	}
	if err := aggregateReceiptStatus(ctx, tx, chat, id); err != nil {
		return err
	}
	status := MessageStatusSent
	switch web.GetStatus() {
	case waWeb.WebMessageInfo_READ, waWeb.WebMessageInfo_PLAYED:
		status = MessageStatusRead
	case waWeb.WebMessageInfo_DELIVERY_ACK:
		status = MessageStatusDelivered
	}
	_, err := tx.ExecContext(ctx, `UPDATE whatsdown_messages SET status = MAX(status, ?) WHERE chat_jid = ? AND id = ? AND from_me = 1`, status, chat, id)
	return err
}

// Merge server-provided LID/phone mappings without guessing identity or status.
func mergeReceiptIdentitiesTx(ctx context.Context, tx execer, chat, id string) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO whatsdown_message_recipients
  (chat_jid, message_id, recipient_jid, expected, status, delivered_at, read_at)
 SELECT r.chat_jid, r.message_id, lm.pn || '@s.whatsapp.net', r.expected, r.status, r.delivered_at, r.read_at
 FROM whatsdown_message_recipients r JOIN whatsmeow_lid_map lm ON r.recipient_jid = lm.lid || '@lid'
 WHERE r.chat_jid = ? AND r.message_id = ? AND lm.pn <> ''
 ON CONFLICT(chat_jid, message_id, recipient_jid) DO UPDATE SET
 expected = MAX(expected, excluded.expected), status = MAX(status, excluded.status),
 delivered_at = CASE WHEN excluded.delivered_at > 0 AND (delivered_at = 0 OR excluded.delivered_at < delivered_at) THEN excluded.delivered_at ELSE delivered_at END,
 read_at = CASE WHEN excluded.read_at > 0 AND (read_at = 0 OR excluded.read_at < read_at) THEN excluded.read_at ELSE read_at END`, chat, id)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `DELETE FROM whatsdown_message_recipients WHERE chat_jid = ? AND message_id = ?
 AND recipient_jid IN (SELECT lid || '@lid' FROM whatsmeow_lid_map WHERE pn <> '')`, chat, id)
	return err
}
