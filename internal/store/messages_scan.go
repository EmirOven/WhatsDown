package store

import (
	"context"
	"database/sql"
	"strings"
	"time"
)

const selectMessagesSQL = `
	SELECT 
		m.chat_jid,
		m.id,
		m.sender_jid,
		COALESCE(
			NULLIF(cnt.full_name, ''),
			NULLIF(cnt_lid.full_name, ''),
			NULLIF(cnt.first_name, ''),
			NULLIF(cnt_lid.first_name, ''),
			NULLIF(cnt.business_name, ''),
			NULLIF(cnt_lid.business_name, ''),
			NULLIF(cnt.push_name, ''),
			NULLIF(cnt_lid.push_name, ''),
			NULLIF(m.sender_name, ''),
			CASE 
				WHEN m.sender_jid LIKE '%@s.whatsapp.net' THEN '+' || SUBSTR(m.sender_jid, 1, INSTR(m.sender_jid, '@') - 1)
				WHEN lm.pn IS NOT NULL AND lm.pn <> '' THEN '+' || lm.pn
				ELSE 'WhatsApp User'
			END
		) AS resolved_sender_name,
		m.from_me,
		m.timestamp,
		m.kind,
		m.text,
		m.edited,
		m.raw,
		m.starred,
		m.status,
		CASE 
			WHEN (cnt.full_name IS NOT NULL AND cnt.full_name <> '')
			  OR (cnt_lid.full_name IS NOT NULL AND cnt_lid.full_name <> '')
			  OR (cnt.first_name IS NOT NULL AND cnt.first_name <> '')
			  OR (cnt_lid.first_name IS NOT NULL AND cnt_lid.first_name <> '')
			  OR (cnt.business_name IS NOT NULL AND cnt.business_name <> '')
			  OR (cnt_lid.business_name IS NOT NULL AND cnt_lid.business_name <> '')
			THEN 1
			ELSE 0
		END AS sender_saved
	FROM whatsdown_messages m
	LEFT JOIN whatsmeow_contacts cnt ON cnt.their_jid = m.sender_jid
	LEFT JOIN whatsmeow_lid_map lm ON (lm.lid || '@lid') = m.sender_jid
	LEFT JOIN whatsmeow_contacts cnt_lid ON cnt_lid.their_jid = (lm.pn || '@s.whatsapp.net')`

func scanStoredMessages(rows *sql.Rows) ([]StoredMessage, error) {
	defer rows.Close()
	var msgs []StoredMessage
	for rows.Next() {
		var (
			m              StoredMessage
			ts             int64
			fromMe, edited bool
			starred        bool
			senderSaved    bool
		)
		if err := rows.Scan(
			&m.ChatJID, &m.ID, &m.SenderJID, &m.SenderName,
			&fromMe, &ts, &m.Kind, &m.Text,
			&edited, &m.Raw, &starred, &m.Status, &senderSaved,
		); err != nil {
			return nil, err
		}
		m.FromMe = fromMe
		m.Edited = edited
		m.Starred = starred
		m.SenderSaved = senderSaved
		m.Timestamp = time.Unix(ts, 0)
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

// Close the message cursor before fetching reactions, including on single-connection stores.
func (s *MessageStore) scanMessagesWithReactions(ctx context.Context, rows *sql.Rows) ([]StoredMessage, error) {
	messages, err := scanStoredMessages(rows)
	if err != nil {
		return nil, err
	}
	type key struct{ chat, id string }
	indices := make(map[key][]int, len(messages))
	for i, msg := range messages {
		k := key{msg.ChatJID, msg.ID}
		indices[k] = append(indices[k], i)
	}
	// Keep batches below SQLite's bind parameter limit and only load reactions
	// for the messages requested, rather than the whole chat history.
	for start := 0; start < len(messages); start += 200 {
		end := start + 200
		if end > len(messages) {
			end = len(messages)
		}
		var clauses []string
		var args []any
		for _, msg := range messages[start:end] {
			clauses = append(clauses, "(chat_jid = ? AND message_id = ?)")
			args = append(args, msg.ChatJID, msg.ID)
		}
		reactions, err := s.db.QueryContext(ctx, `SELECT chat_jid, message_id, sender_jid, sender_name, emoji, from_me, timestamp FROM whatsdown_reactions WHERE `+strings.Join(clauses, " OR ")+` ORDER BY timestamp ASC`, args...)
		if err != nil {
			return nil, err
		}
		for reactions.Next() {
			var r Reaction
			var ts int64
			if err := reactions.Scan(&r.ChatJID, &r.MessageID, &r.SenderJID, &r.SenderName, &r.Emoji, &r.FromMe, &ts); err != nil {
				reactions.Close()
				return nil, err
			}
			r.Timestamp = time.Unix(ts, 0)
			for _, i := range indices[key{r.ChatJID, r.MessageID}] {
				messages[i].Reactions = append(messages[i].Reactions, r)
			}
		}
		err = reactions.Err()
		reactions.Close()
		if err != nil {
			return nil, err
		}
	}
	return messages, nil
}
