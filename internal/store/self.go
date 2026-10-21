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

	s.selfMu.Lock()
	defer s.selfMu.Unlock()

	var devJID, devLID string
	_ = s.db.QueryRowContext(ctx, `SELECT jid, COALESCE(lid, '') FROM whatsmeow_device LIMIT 1`).Scan(&devJID, &devLID)
	clean := func(j, server string) string {
		if at := strings.Index(j, "@"); at != -1 {
			j = j[:at]
		}
		if colon := strings.Index(j, ":"); colon != -1 {
			j = j[:colon]
		}
		if j != "" {
			return j + "@" + server
		}
		return ""
	}
	s.selfPN = clean(devJID, "s.whatsapp.net")
	s.selfLID = clean(devLID, "lid")
	return s.selfPN, s.selfLID
}

func (s *MessageStore) IsSelfJID(ctx context.Context, jid string) bool {
	if jid == "" {
		return false
	}
	pn, lid := s.GetSelfJIDs(ctx)
	cleanUser := func(j string) string {
		if at := strings.Index(j, "@"); at != -1 {
			j = j[:at]
		}
		if colon := strings.Index(j, ":"); colon != -1 {
			j = j[:colon]
		}
		return j
	}
	user := cleanUser(jid)
	if pn != "" && cleanUser(pn) == user {
		return true
	}
	if lid != "" && cleanUser(lid) == user {
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
