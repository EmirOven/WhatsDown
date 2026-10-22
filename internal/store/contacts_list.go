package store

import (
	"context"
	"database/sql"
	"strings"
)

// ListAllContacts returns all known 1-on-1 user contacts from whatsmeow_contacts.
func (s *MessageStore) ListAllContacts(ctx context.Context) ([]Chat, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT their_jid, first_name, full_name, push_name, business_name
		FROM whatsmeow_contacts
		WHERE their_jid LIKE '%@s.whatsapp.net'
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Chat
	for rows.Next() {
		var (
			jid            string
			fn, fln, pn, bn sql.NullString
		)
		if err := rows.Scan(&jid, &fn, &fln, &pn, &bn); err != nil {
			continue
		}
		if idx := strings.Index(jid, ":"); idx != -1 {
			at := strings.Index(jid, "@")
			if at > idx {
				jid = jid[:idx] + jid[at:]
			}
		}
		name := fln.String
		if name == "" {
			name = fn.String
		}
		if name == "" {
			name = bn.String
		}
		if name == "" {
			name = pn.String
		}
		name = strings.TrimPrefix(strings.TrimSpace(name), "@")
		result = append(result, Chat{
			JID:  jid,
			Name: name,
		})
	}
	return result, rows.Err()
}
