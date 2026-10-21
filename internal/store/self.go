package store

import (
	"context"
	"strings"
)

func (s *MessageStore) GetSelfJIDs(ctx context.Context) (pn string, lid string) {
	s.selfMu.RLock()
	if s.selfPN != "" || s.selfLID != "" {
		p, l := s.selfPN, s.selfLID
		s.selfMu.RUnlock()
		return p, l
	}
	s.selfMu.RUnlock()

	var ourJID string
	_ = s.db.QueryRowContext(ctx, `SELECT our_jid FROM whatsmeow_contacts WHERE our_jid != '' LIMIT 1`).Scan(&ourJID)

	s.selfMu.Lock()
	defer s.selfMu.Unlock()
	if strings.Contains(ourJID, "@lid") {
		s.selfLID = ourJID
	} else if ourJID != "" {
		s.selfPN = ourJID
	}
	return s.selfPN, s.selfLID
}

func (s *MessageStore) IsSelfJID(ctx context.Context, jid string) bool {
	if jid == "" {
		return false
	}
	pn, lid := s.GetSelfJIDs(ctx)
	clean := func(j string) string {
		if at := strings.Index(j, "@"); at != -1 {
			j = j[:at]
		}
		if colon := strings.Index(j, ":"); colon != -1 {
			j = j[:colon]
		}
		return j
	}
	user := clean(jid)
	if pn != "" && clean(pn) == user {
		return true
	}
	if lid != "" && clean(lid) == user {
		return true
	}
	return false
}

func (s *MessageStore) GetCanonicalSelfJID(ctx context.Context) string {
	pn, lid := s.GetSelfJIDs(ctx)
	if pn != "" {
		return pn
	}
	if lid != "" {
		return lid
	}
	return "me"
}

func (s *MessageStore) cleanupSelfChats(ctx context.Context) {
	pn, lid := s.GetSelfJIDs(ctx)
	if pn == "" && lid == "" {
		return
	}
	canonical := s.GetCanonicalSelfJID(ctx)
	other := lid
	if canonical == lid {
		other = pn
	}
	if other == "" || canonical == other {
		return
	}
	_, _ = s.db.ExecContext(ctx, `UPDATE OR IGNORE whatsdown_messages SET chat_jid = ? WHERE chat_jid = ?`, canonical, other)
	_, _ = s.db.ExecContext(ctx, `DELETE FROM whatsdown_chats WHERE jid = ?`, other)
}

func (s *MessageStore) ResolveLIDtoPN(ctx context.Context, jid string) string {
	lidPart := jid
	if at := strings.Index(lidPart, "@"); at != -1 {
		lidPart = lidPart[:at]
	}
	var pn string
	_ = s.db.QueryRowContext(ctx, `SELECT pn FROM whatsmeow_lid_map WHERE lid = ?`, lidPart).Scan(&pn)
	if pn != "" {
		return pn + "@s.whatsapp.net"
	}
	return ""
}
