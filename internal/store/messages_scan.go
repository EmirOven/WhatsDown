package store

import (
	"database/sql"
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
