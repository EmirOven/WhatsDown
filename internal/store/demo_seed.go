package store

import (
	"context"
	"time"
)

// SeedDemoData populates the database with rich demo chats, messages, polls, reactions, wallpapers, and calls.
func (s *MessageStore) SeedDemoData(ctx context.Context) error {
	s.SetDemo(true)
	var count int
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM whatsdown_chats`).Scan(&count)
	if count > 0 {
		return nil
	}

	now := time.Now()
	if err := s.seedDemoContactsAndChats(ctx, now); err != nil {
		return err
	}
	if err := s.seedDemoMessages(ctx, now); err != nil {
		return err
	}
	if err := s.seedDemoExtras(ctx, now); err != nil {
		return err
	}
	return nil
}

func (s *MessageStore) seedDemoContactsAndChats(ctx context.Context, now time.Time) error {
	contacts := []struct{ jid, first, full string }{
		{"15550100001@s.whatsapp.net", "Alice", "Alice Rivera"},
		{"15550100002@s.whatsapp.net", "Marco", "Marco Rossi"},
		{"15550100003@s.whatsapp.net", "Elena", "Elena Rostova"},
		{"15550100004@s.whatsapp.net", "Liam", "Liam O'Connor"},
		{"15550100005@s.whatsapp.net", "Sophia", "Sophia Chen"},
		{"15550100006@s.whatsapp.net", "Noah", "Noah Williams"},
	}
	for _, c := range contacts {
		_, _ = s.db.ExecContext(ctx, `
			INSERT OR REPLACE INTO whatsmeow_contacts (our_jid, their_jid, first_name, full_name, push_name)
			VALUES (?, ?, ?, ?, ?)`, DemoSelfJID, c.jid, c.first, c.full, c.full)
	}

	chats := []struct {
		jid, name, parent               string
		minsAgo, unread, arch, pin, mut int
		isParent, isDefSub              int
	}{
		{"120363000000000001@g.us", "WhatsDown Core Team", "", 3, 0, 0, 1, 0, 0, 0},
		{"15550100001@s.whatsapp.net", "Alice Rivera", "", 8, 2, 0, 1, 0, 0, 0},
		{"15550100002@s.whatsapp.net", "Marco Rossi", "", 25, 0, 0, 0, 0, 0, 0},
		{"120363000000000010@g.us", "Linux Desktop Community", "", 40, 0, 0, 0, 0, 1, 0},
		{"120363000000000011@g.us", "Announcements", "120363000000000010@g.us", 45, 0, 0, 0, 0, 0, 1},
		{"120363000000000012@g.us", "GTK4 & Libadwaita", "120363000000000010@g.us", 40, 1, 0, 0, 0, 0, 0},
		{"120363000000000002@g.us", "Weekend Hiking Crew", "", 180, 0, 0, 0, 1, 0, 0},
		{"15550100003@s.whatsapp.net", "Elena Rostova", "", 420, 0, 0, 0, 0, 0, 0},
		{"120363000000000099@g.us", "Old Project Archive", "", 2880, 0, 1, 0, 1, 0, 0},
	}
	for _, ch := range chats {
		ts := now.Add(-time.Duration(ch.minsAgo) * time.Minute).Unix()
		_, err := s.db.ExecContext(ctx, `
			INSERT OR REPLACE INTO whatsdown_chats
			(jid, name, last_message_at, unread_count, archived, pinned, muted, parent_jid, is_parent, is_default_sub_group)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			ch.jid, ch.name, ts, ch.unread, ch.arch, ch.pin, ch.mut, ch.parent, ch.isParent, ch.isDefSub)
		if err != nil {
			return err
		}
	}
	return nil
}
