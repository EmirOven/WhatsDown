package main

// Chat and message storage.
//
// whatsmeow's sqlstore only keeps the login session (keys, contacts, app
// state). Chats and messages arrive as events and are persisted here, in our
// own whatsdown_* tables inside the same SQLite file.

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/proto/waHistorySync"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

// migrations are applied in order; the number applied so far is kept in
// whatsdown_version. Never edit an existing entry, only append new ones.
var migrations = []string{
	// 1: chats and messages
	`
	CREATE TABLE whatsdown_chats (
		jid             TEXT PRIMARY KEY,
		name            TEXT    NOT NULL DEFAULT '',
		last_message_at INTEGER NOT NULL DEFAULT 0, -- unix seconds
		unread_count    INTEGER NOT NULL DEFAULT 0,
		archived        INTEGER NOT NULL DEFAULT 0,
		pinned          INTEGER NOT NULL DEFAULT 0
	);

	CREATE TABLE whatsdown_messages (
		chat_jid    TEXT    NOT NULL REFERENCES whatsdown_chats(jid) ON DELETE CASCADE,
		id          TEXT    NOT NULL,
		sender_jid  TEXT    NOT NULL,
		sender_name TEXT    NOT NULL DEFAULT '',
		from_me     INTEGER NOT NULL,
		timestamp   INTEGER NOT NULL, -- unix seconds
		kind        TEXT    NOT NULL, -- see messageKind* constants
		text        TEXT    NOT NULL DEFAULT '',
		edited      INTEGER NOT NULL DEFAULT 0,
		raw         BLOB,             -- marshaled waE2E.Message, for media etc. later
		PRIMARY KEY (chat_jid, id)
	);

	CREATE INDEX whatsdown_messages_chat_time ON whatsdown_messages (chat_jid, timestamp);
	`,

	// 2: ensure contacts tables exist for contact name resolution
	`
	CREATE TABLE IF NOT EXISTS whatsmeow_contacts (
		our_jid        TEXT NOT NULL DEFAULT '',
		their_jid      TEXT NOT NULL DEFAULT '',
		first_name     TEXT NOT NULL DEFAULT '',
		full_name      TEXT NOT NULL DEFAULT '',
		push_name      TEXT NOT NULL DEFAULT '',
		business_name  TEXT NOT NULL DEFAULT '',
		redacted_phone TEXT NOT NULL DEFAULT '',
		PRIMARY KEY (our_jid, their_jid)
	);

	CREATE TABLE IF NOT EXISTS whatsmeow_lid_map (
		lid TEXT PRIMARY KEY,
		pn  TEXT
	);
	`,

	// 3: add starred message support
	`
	ALTER TABLE whatsdown_messages ADD COLUMN starred INTEGER NOT NULL DEFAULT 0;
	CREATE INDEX IF NOT EXISTS whatsdown_messages_starred ON whatsdown_messages (chat_jid, starred);
	`,

	// 4: poll votes
	`
	CREATE TABLE IF NOT EXISTS whatsdown_poll_votes (
		chat_jid    TEXT NOT NULL,
		message_id  TEXT NOT NULL,
		option_name TEXT NOT NULL,
		PRIMARY KEY (chat_jid, message_id, option_name)
	);
	`,

	// 5: poll votes with voter_jid support
	`
	CREATE TABLE IF NOT EXISTS whatsdown_poll_votes_new (
		chat_jid    TEXT NOT NULL,
		message_id  TEXT NOT NULL,
		voter_jid   TEXT NOT NULL,
		option_name TEXT NOT NULL,
		PRIMARY KEY (chat_jid, message_id, voter_jid, option_name)
	);
	INSERT OR IGNORE INTO whatsdown_poll_votes_new (chat_jid, message_id, voter_jid, option_name)
		SELECT chat_jid, message_id, 'me', option_name FROM whatsdown_poll_votes;
	DROP TABLE whatsdown_poll_votes;
	ALTER TABLE whatsdown_poll_votes_new RENAME TO whatsdown_poll_votes;
	CREATE INDEX IF NOT EXISTS idx_poll_votes_msg ON whatsdown_poll_votes (chat_jid, message_id);
	`,

	// 6: mute chat support
	`
	ALTER TABLE whatsdown_chats ADD COLUMN muted INTEGER NOT NULL DEFAULT 0;
	`,
}

// Values of whatsdown_messages.kind.
const (
	messageKindText     = "text"
	messageKindImage    = "image"
	messageKindVideo    = "video"
	messageKindAudio    = "audio"
	messageKindVoice    = "voice"
	messageKindDocument = "document"
	messageKindSticker  = "sticker"
	messageKindLocation = "location"
	messageKindContact  = "contact"
	messageKindPoll     = "poll"
	messageKindDeleted  = "deleted"
)

type messageStore struct {
	db      *sql.DB
	selfMu  sync.RWMutex
	selfPN  string
	selfLID string
}

// execer is satisfied by both *sql.DB and *sql.Tx.
type execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func openMessageStore(ctx context.Context, db *sql.DB) (*messageStore, error) {
	s := &messageStore{db: db}
	if err := s.migrate(ctx); err != nil {
		return nil, fmt.Errorf("migrating message store: %w", err)
	}
	s.cleanupSelfChats(ctx)
	return s, nil
}

func (s *messageStore) migrate(ctx context.Context) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS whatsdown_version (version INTEGER NOT NULL)`); err != nil {
		return err
	}
	var version int
	err = tx.QueryRowContext(ctx, `SELECT version FROM whatsdown_version`).Scan(&version)
	switch {
	case err == sql.ErrNoRows:
		if _, err := tx.ExecContext(ctx, `INSERT INTO whatsdown_version (version) VALUES (0)`); err != nil {
			return err
		}
	case err != nil:
		return err
	}

	for i := version; i < len(migrations); i++ {
		if _, err := tx.ExecContext(ctx, migrations[i]); err != nil {
			return fmt.Errorf("migration %d: %w", i+1, err)
		}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE whatsdown_version SET version = ?`, len(migrations)); err != nil {
		return err
	}
	return tx.Commit()
}

// saveHistorySync stores one history sync chunk (sent by the phone after
// pairing) in a single transaction and returns how many messages were stored.
func (s *messageStore) saveHistorySync(ctx context.Context, client *whatsmeow.Client, data *waHistorySync.HistorySync) (int, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	stored := 0
	for _, pn := range data.GetPushnames() {
		if pn.GetID() != "" && pn.GetPushname() != "" {
			_, _ = tx.ExecContext(ctx, `
				INSERT INTO whatsmeow_contacts (our_jid, their_jid, push_name)
				VALUES ('', ?, ?)
				ON CONFLICT (our_jid, their_jid) DO UPDATE SET
					push_name = CASE WHEN push_name = '' OR push_name IS NULL THEN excluded.push_name ELSE push_name END`,
				pn.GetID(), pn.GetPushname())
		}
	}

	for _, conv := range data.GetConversations() {
		chatJID, err := types.ParseJID(conv.GetID())
		if err != nil {
			continue
		}

		name := conv.GetDisplayName()
		if name == "" {
			name = conv.GetName()
		}
		if name == "" {
			// For DMs, resolve phone contact name from whatsmeow_contacts or mapped LID
			_ = tx.QueryRowContext(ctx, `
				SELECT COALESCE(
					NULLIF(cnt.full_name, ''),
					NULLIF(cnt_lid.full_name, ''),
					NULLIF(cnt.first_name, ''),
					NULLIF(cnt_lid.first_name, ''),
					NULLIF(cnt.business_name, ''),
					NULLIF(cnt_lid.business_name, ''),
					NULLIF(cnt.push_name, ''),
					NULLIF(cnt_lid.push_name, ''),
					''
				)
				FROM (SELECT ? AS jid) t
				LEFT JOIN whatsmeow_contacts cnt ON cnt.their_jid = t.jid
				LEFT JOIN whatsmeow_lid_map lm ON (lm.lid || '@lid') = t.jid
				LEFT JOIN whatsmeow_contacts cnt_lid ON cnt_lid.their_jid = (lm.pn || '@s.whatsapp.net')`,
				chatJID.String()).Scan(&name)
		}
		_, err = tx.ExecContext(ctx, `
			INSERT INTO whatsdown_chats (jid, name, last_message_at, unread_count, archived, pinned)
			VALUES (?, ?, ?, ?, ?, ?)
			ON CONFLICT (jid) DO UPDATE SET
				name            = CASE WHEN excluded.name <> '' THEN excluded.name ELSE name END,
				last_message_at = MAX(last_message_at, excluded.last_message_at),
				unread_count    = excluded.unread_count,
				archived        = excluded.archived,
				pinned          = excluded.pinned`,
			chatJID.String(), name, int64(conv.GetConversationTimestamp()),
			conv.GetUnreadCount(), conv.GetArchived(), conv.GetPinned() > 0)
		if err != nil {
			return 0, err
		}

		for _, hsm := range conv.GetMessages() {
			evt, err := client.ParseWebMessage(chatJID, hsm.GetMessage())
			if err != nil {
				continue
			}
			isStarred := hsm.GetMessage().GetStarred()
			inserted, err := s.saveMessageTxWithStar(ctx, tx, evt, isStarred)
			if err != nil {
				return 0, err
			}
			if inserted {
				stored++
			}
		}
	}

	return stored, tx.Commit()
}

// saveLiveMessage stores a message received while connected (including
// messages delivered right after reconnecting), applying edits and deletions.
func (s *messageStore) saveLiveMessage(ctx context.Context, evt *events.Message) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	inserted, err := s.saveMessageTx(ctx, tx, evt)
	if err != nil {
		return err
	}
	chatJID := evt.Info.Chat.ToNonAD().String()
	isSelf := s.isSelfJID(ctx, chatJID)
	if isSelf {
		if canonical := s.getCanonicalSelfJID(ctx); canonical != "" {
			chatJID = canonical
		}
	}
	if inserted && !evt.Info.IsFromMe && !isSelf {
		res, err := tx.ExecContext(ctx,
			`UPDATE whatsdown_chats SET unread_count = unread_count + 1, last_message_at = MAX(last_message_at, ?) WHERE jid = ?`,
			evt.Info.Timestamp.Unix(), chatJID)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			_, err = tx.ExecContext(ctx,
				`INSERT INTO whatsdown_chats (jid, name, last_message_at, unread_count, archived, pinned, muted) VALUES (?, ?, ?, 1, 0, 0, 0)`,
				chatJID, evt.Info.PushName, evt.Info.Timestamp.Unix())
			if err != nil {
				return err
			}
		}
	} else if inserted && isSelf {
		_, err = tx.ExecContext(ctx,
			`UPDATE whatsdown_chats SET last_message_at = MAX(last_message_at, ?) WHERE jid = ?`,
			evt.Info.Timestamp.Unix(), chatJID)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

// saveMessageTx stores a single message.
func (s *messageStore) saveMessageTx(ctx context.Context, tx execer, evt *events.Message) (bool, error) {
	return s.saveMessageTxWithStar(ctx, tx, evt, false)
}

// saveMessageTxWithStar stores a single message with an initial starred state.
// Edits and deletions update the original message instead. Messages without
// user-visible content (reactions, key distribution, …) are skipped.
// Returns whether a new row was inserted.
func (s *messageStore) saveMessageTxWithStar(ctx context.Context, tx execer, evt *events.Message, starred bool) (bool, error) {
	chat := evt.Info.Chat.ToNonAD().String()
	isSelf := s.isSelfJID(ctx, chat)
	if isSelf {
		if canonical := s.getCanonicalSelfJID(ctx); canonical != "" {
			chat = canonical
		}
	}

	if pm := evt.Message.GetProtocolMessage(); pm != nil {
		targetID := pm.GetKey().GetID()
		switch pm.GetType() {
		case waE2E.ProtocolMessage_REVOKE:
			_, err := tx.ExecContext(ctx,
				`UPDATE whatsdown_messages SET kind = ?, text = '', raw = NULL WHERE chat_jid = ? AND id = ?`,
				messageKindDeleted, chat, targetID)
			return false, err
		case waE2E.ProtocolMessage_MESSAGE_EDIT:
			if _, text, ok := describeMessage(pm.GetEditedMessage()); ok {
				raw, _ := proto.Marshal(pm.GetEditedMessage())
				_, err := tx.ExecContext(ctx,
					`UPDATE whatsdown_messages SET text = ?, raw = ?, edited = 1 WHERE chat_jid = ? AND id = ?`,
					text, raw, chat, targetID)
				return false, err
			}
		}
		return false, nil
	}

	kind, text, ok := describeMessage(evt.Message)
	if !ok {
		return false, nil
	}
	raw, err := proto.Marshal(evt.Message)
	if err != nil {
		raw = nil
	}

	// Use the sender's push name as a fallback name for direct chats.
	nameHint := ""
	if !evt.Info.IsGroup && !evt.Info.IsFromMe && !isSelf {
		nameHint = evt.Info.PushName
	}
	ts := evt.Info.Timestamp.Unix()
	_, err = tx.ExecContext(ctx, `
		INSERT INTO whatsdown_chats (jid, name, last_message_at) VALUES (?, ?, ?)
		ON CONFLICT (jid) DO UPDATE SET
			name            = CASE WHEN name = '' THEN excluded.name ELSE name END,
			last_message_at = MAX(last_message_at, excluded.last_message_at)`,
		chat, nameHint, ts)
	if err != nil {
		return false, err
	}

	starredInt := 0
	if starred {
		starredInt = 1
	}

	fromMeVal := 0
	if evt.Info.IsFromMe || isSelf {
		fromMeVal = 1
	}

	res, err := tx.ExecContext(ctx, `
		INSERT INTO whatsdown_messages (chat_jid, id, sender_jid, sender_name, from_me, timestamp, kind, text, raw, starred)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (chat_jid, id) DO UPDATE SET
			starred = CASE WHEN excluded.starred = 1 THEN 1 ELSE whatsdown_messages.starred END,
			from_me = CASE WHEN excluded.from_me = 1 THEN 1 ELSE whatsdown_messages.from_me END`,
		chat, evt.Info.ID, evt.Info.Sender.ToNonAD().String(), evt.Info.PushName,
		fromMeVal, ts, kind, text, raw, starredInt)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// markChatRead resets the unread counter, e.g. when the chat was read on the
// phone.
func (s *messageStore) markChatRead(ctx context.Context, chat types.JID) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE whatsdown_chats SET unread_count = 0 WHERE jid = ?`, chat.ToNonAD().String())
	return err
}

// describeMessage returns the kind and display text of a message, or ok=false
// for messages that have no user-visible content.
func describeMessage(m *waE2E.Message) (kind, text string, ok bool) {
	switch {
	case m == nil:
		return "", "", false
	case m.GetConversation() != "":
		return messageKindText, m.GetConversation(), true
	case m.GetExtendedTextMessage() != nil:
		return messageKindText, m.GetExtendedTextMessage().GetText(), true
	case m.GetImageMessage() != nil:
		return messageKindImage, m.GetImageMessage().GetCaption(), true
	case m.GetVideoMessage() != nil:
		return messageKindVideo, m.GetVideoMessage().GetCaption(), true
	case m.GetAudioMessage() != nil:
		if m.GetAudioMessage().GetPTT() {
			return messageKindVoice, "", true
		}
		return messageKindAudio, "", true
	case m.GetDocumentMessage() != nil:
		doc := m.GetDocumentMessage()
		text := doc.GetCaption()
		if text == "" {
			text = doc.GetFileName()
		}
		if text == "" {
			text = doc.GetTitle()
		}
		return messageKindDocument, text, true
	case m.GetStickerMessage() != nil:
		return messageKindSticker, "", true
	case m.GetLocationMessage() != nil:
		loc := m.GetLocationMessage()
		text := loc.GetName()
		if text == "" {
			text = loc.GetAddress()
		}
		return messageKindLocation, text, true
	case m.GetLiveLocationMessage() != nil:
		return messageKindLocation, m.GetLiveLocationMessage().GetCaption(), true
	case m.GetContactMessage() != nil:
		return messageKindContact, m.GetContactMessage().GetDisplayName(), true
	case m.GetContactsArrayMessage() != nil:
		return messageKindContact, m.GetContactsArrayMessage().GetDisplayName(), true
	}
	for _, poll := range []*waE2E.PollCreationMessage{
		m.GetPollCreationMessage(), m.GetPollCreationMessageV2(), m.GetPollCreationMessageV3(),
		m.GetPollCreationMessageV5(), m.GetPollCreationMessageV6(),
	} {
		if poll != nil {
			return messageKindPoll, poll.GetName(), true
		}
	}
	return "", "", false
}

// Chat is a row of whatsdown_chats.
type Chat struct {
	JID           string
	Name          string
	LastMessageAt time.Time
	UnreadCount   int
	Archived      bool
	Pinned        bool
	Muted         bool
}

// StoredMessage is a row of whatsdown_messages (without the raw protobuf).
type StoredMessage struct {
	ChatJID     string
	ID          string
	SenderJID   string
	SenderName  string
	FromMe      bool
	Timestamp   time.Time
	Kind        string
	Text        string
	Edited      bool
	Raw         []byte
	Starred     bool
	SenderSaved bool
}

// listChats returns all unarchived chats, pinned first, then most recently active.
// For DMs, contact names saved on the phone (whatsmeow_contacts) are prioritized over JIDs.
func (s *messageStore) listChats(ctx context.Context) ([]Chat, error) {
	return s.fetchChats(ctx, false)
}

// listArchivedChats returns all archived chats, pinned first, then most recently active.
func (s *messageStore) listArchivedChats(ctx context.Context) ([]Chat, error) {
	return s.fetchChats(ctx, true)
}

func (s *messageStore) fetchChats(ctx context.Context, archived bool) ([]Chat, error) {
	archivedVal := 0
	if archived {
		archivedVal = 1
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT 
			c.jid,
			CASE 
				WHEN c.jid LIKE '%@g.us' THEN COALESCE(NULLIF(c.name, ''), 'Group')
				WHEN c.jid LIKE '%@newsletter' THEN COALESCE(NULLIF(c.name, ''), 'Channel')
				WHEN c.jid LIKE '%@broadcast' THEN 'Status Broadcast'
				ELSE COALESCE(
					NULLIF(cnt.full_name, ''),
					NULLIF(cnt_lid.full_name, ''),
					NULLIF(cnt.first_name, ''),
					NULLIF(cnt_lid.first_name, ''),
					NULLIF(cnt.business_name, ''),
					NULLIF(cnt_lid.business_name, ''),
					NULLIF(cnt.push_name, ''),
					NULLIF(cnt_lid.push_name, ''),
					NULLIF(c.name, ''),
					CASE WHEN lm.pn IS NOT NULL AND lm.pn <> '' THEN '+' || lm.pn ELSE NULL END,
					c.jid
				)
			END AS resolved_name,
			c.last_message_at,
			c.unread_count,
			c.archived,
			c.pinned,
			c.muted
		FROM whatsdown_chats c
		LEFT JOIN whatsmeow_contacts cnt ON cnt.their_jid = c.jid
		LEFT JOIN whatsmeow_lid_map lm ON (lm.lid || '@lid') = c.jid
		LEFT JOIN whatsmeow_contacts cnt_lid ON cnt_lid.their_jid = (lm.pn || '@s.whatsapp.net')
		WHERE c.archived = ?
		ORDER BY c.pinned DESC, c.last_message_at DESC`, archivedVal)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chats []Chat
	for rows.Next() {
		var c Chat
		var ts int64
		if err := rows.Scan(&c.JID, &c.Name, &ts, &c.UnreadCount, &c.Archived, &c.Pinned, &c.Muted); err != nil {
			return nil, err
		}
		c.LastMessageAt = time.Unix(ts, 0)
		chats = append(chats, c)
	}
	return chats, rows.Err()
}

// setChatPinned updates the pinned status of a chat.
func (s *messageStore) setChatPinned(ctx context.Context, jid string, pinned bool) error {
	pinnedVal := 0
	if pinned {
		pinnedVal = 1
	}
	_, err := s.db.ExecContext(ctx, `UPDATE whatsdown_chats SET pinned = ? WHERE jid = ?`, pinnedVal, jid)
	return err
}

// setChatArchived updates the archived status of a chat.
func (s *messageStore) setChatArchived(ctx context.Context, jid string, archived bool) error {
	archivedVal := 0
	if archived {
		archivedVal = 1
	}
	_, err := s.db.ExecContext(ctx, `UPDATE whatsdown_chats SET archived = ? WHERE jid = ?`, archivedVal, jid)
	return err
}

// setChatMuted updates the muted status of a chat.
func (s *messageStore) setChatMuted(ctx context.Context, jid string, muted bool) error {
	mutedVal := 0
	if muted {
		mutedVal = 1
	}
	_, err := s.db.ExecContext(ctx, `UPDATE whatsdown_chats SET muted = ? WHERE jid = ?`, mutedVal, jid)
	return err
}

// getChatOrContact retrieves a Chat by JID, resolving contact names and metadata.
func (s *messageStore) getChatOrContact(ctx context.Context, jid string) (Chat, error) {
	var name string
	var lastMsgAt int64
	var unread int
	var archived, pinned, muted bool

	err := s.db.QueryRowContext(ctx, `
		SELECT
			COALESCE(
				NULLIF(cnt.full_name, ''),
				NULLIF(cnt_lid.full_name, ''),
				NULLIF(cnt.first_name, ''),
				NULLIF(cnt_lid.first_name, ''),
				NULLIF(cnt.business_name, ''),
				NULLIF(cnt_lid.business_name, ''),
				NULLIF(cnt.push_name, ''),
				NULLIF(cnt_lid.push_name, ''),
				NULLIF(c.name, ''),
				CASE WHEN lm.pn IS NOT NULL AND lm.pn <> '' THEN '+' || lm.pn ELSE NULL END,
				''
			) AS resolved_name,
			COALESCE(c.last_message_at, 0),
			COALESCE(c.unread_count, 0),
			COALESCE(c.archived, 0),
			COALESCE(c.pinned, 0),
			COALESCE(c.muted, 0)
		FROM (SELECT ? AS jid) t
		LEFT JOIN whatsdown_chats c ON c.jid = t.jid
		LEFT JOIN whatsmeow_contacts cnt ON cnt.their_jid = t.jid
		LEFT JOIN whatsmeow_lid_map lm ON (lm.lid || '@lid') = t.jid
		LEFT JOIN whatsmeow_contacts cnt_lid ON cnt_lid.their_jid = (lm.pn || '@s.whatsapp.net')`,
		jid).Scan(&name, &lastMsgAt, &unread, &archived, &pinned, &muted)
	if err != nil {
		return Chat{JID: jid}, err
	}
	return Chat{
		JID:           jid,
		Name:          name,
		LastMessageAt: time.Unix(lastMsgAt, 0),
		UnreadCount:   unread,
		Archived:      archived,
		Pinned:        pinned,
		Muted:         muted,
	}, nil
}

// getSelfJIDs retrieves own phone number JID and own LID from whatsmeow_device.
func (s *messageStore) getSelfJIDs(ctx context.Context) (pn string, lid string) {
	s.selfMu.RLock()
	pn = s.selfPN
	lid = s.selfLID
	s.selfMu.RUnlock()
	if pn != "" && lid != "" {
		return pn, lid
	}

	var rawPN, rawLID string
	err := s.db.QueryRowContext(ctx, `SELECT jid, lid FROM whatsmeow_device LIMIT 1`).Scan(&rawPN, &rawLID)
	if err == nil {
		rawPN = strings.Split(rawPN, ":")[0]
		if rawPN != "" && !strings.Contains(rawPN, "@") {
			rawPN = rawPN + "@s.whatsapp.net"
		}
		rawLID = strings.Split(rawLID, ":")[0]
		if rawLID != "" && !strings.Contains(rawLID, "@") {
			rawLID = rawLID + "@lid"
		}
		s.selfMu.Lock()
		s.selfPN = rawPN
		s.selfLID = rawLID
		pn = s.selfPN
		lid = s.selfLID
		s.selfMu.Unlock()
	}
	return
}

// isSelfJID checks whether the given JID represents the logged-in user.
func (s *messageStore) isSelfJID(ctx context.Context, jid string) bool {
	if jid == "" {
		return false
	}
	parsed, err := types.ParseJID(jid)
	if err != nil {
		return false
	}
	nonAD := parsed.ToNonAD().String()
	user := parsed.User

	pn, lid := s.getSelfJIDs(ctx)
	if pn != "" {
		parsedPN, _ := types.ParseJID(pn)
		if nonAD == parsedPN.ToNonAD().String() || user == parsedPN.User {
			return true
		}
	}
	if lid != "" {
		parsedLID, _ := types.ParseJID(lid)
		if nonAD == parsedLID.ToNonAD().String() || user == parsedLID.User {
			return true
		}
	}
	return false
}

// getCanonicalSelfJID returns the primary JID for the logged-in user.
func (s *messageStore) getCanonicalSelfJID(ctx context.Context) string {
	pn, _ := s.getSelfJIDs(ctx)
	if pn != "" {
		parsedPN, err := types.ParseJID(pn)
		if err == nil {
			return parsedPN.ToNonAD().String()
		}
		return pn
	}
	return ""
}

// cleanupSelfChats merges duplicate self-chats in the database into the canonical self JID.
func (s *messageStore) cleanupSelfChats(ctx context.Context) {
	pn, lid := s.getSelfJIDs(ctx)
	if pn == "" || lid == "" {
		return
	}
	parsedPN, _ := types.ParseJID(pn)
	parsedLID, _ := types.ParseJID(lid)
	pnNonAD := parsedPN.ToNonAD().String()
	lidNonAD := parsedLID.ToNonAD().String()
	if pnNonAD == "" || lidNonAD == "" || pnNonAD == lidNonAD {
		return
	}

	_, _ = s.db.ExecContext(ctx, `
		UPDATE OR IGNORE whatsdown_messages SET chat_jid = ? WHERE chat_jid = ?;
	`, pnNonAD, lidNonAD)
	_, _ = s.db.ExecContext(ctx, `
		DELETE FROM whatsdown_messages WHERE chat_jid = ?;
	`, lidNonAD)
	_, _ = s.db.ExecContext(ctx, `
		DELETE FROM whatsdown_chats WHERE jid = ?;
	`, lidNonAD)
	_, _ = s.db.ExecContext(ctx, `
		UPDATE whatsdown_messages SET from_me = 1 WHERE chat_jid = ?;
	`, pnNonAD)
}

// toggleMessageStarred updates the starred status of a message.
func (s *messageStore) toggleMessageStarred(ctx context.Context, chatJID, id string, starred bool) error {
	starredInt := 0
	if starred {
		starredInt = 1
	}
	_, err := s.db.ExecContext(ctx, `UPDATE whatsdown_messages SET starred = ? WHERE chat_jid = ? AND id = ?`, starredInt, chatJID, id)
	return err
}

func scanStoredMessages(rows *sql.Rows) ([]StoredMessage, error) {
	defer rows.Close()
	var msgs []StoredMessage
	for rows.Next() {
		var m StoredMessage
		var ts int64
		if err := rows.Scan(&m.ChatJID, &m.ID, &m.SenderJID, &m.SenderName, &m.FromMe, &ts, &m.Kind, &m.Text, &m.Edited, &m.Raw, &m.Starred, &m.SenderSaved); err != nil {
			return nil, err
		}
		m.Timestamp = time.Unix(ts, 0)
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

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

// listMediaMessages returns messages in a chat that contain photos or videos.
func (s *messageStore) listMediaMessages(ctx context.Context, chatJID string) ([]StoredMessage, error) {
	query := selectMessagesSQL + `
		WHERE m.chat_jid = ? AND m.kind IN ('image', 'video')
		ORDER BY m.timestamp DESC`
	rows, err := s.db.QueryContext(ctx, query, chatJID)
	if err != nil {
		return nil, err
	}
	return scanStoredMessages(rows)
}

// listAudioMessages returns messages in a chat that contain audio or voice notes.
func (s *messageStore) listAudioMessages(ctx context.Context, chatJID string) ([]StoredMessage, error) {
	query := selectMessagesSQL + `
		WHERE m.chat_jid = ? AND m.kind IN ('audio', 'voice')
		ORDER BY m.timestamp DESC`
	rows, err := s.db.QueryContext(ctx, query, chatJID)
	if err != nil {
		return nil, err
	}
	return scanStoredMessages(rows)
}

// listDocMessages returns documents and other non-media shared files in a chat.
func (s *messageStore) listDocMessages(ctx context.Context, chatJID string) ([]StoredMessage, error) {
	query := selectMessagesSQL + `
		WHERE m.chat_jid = ? AND m.kind NOT IN ('text', 'deleted', 'image', 'video', 'audio', 'voice')
		ORDER BY m.timestamp DESC`
	rows, err := s.db.QueryContext(ctx, query, chatJID)
	if err != nil {
		return nil, err
	}
	return scanStoredMessages(rows)
}

// listStarredMessages returns all starred messages in a chat.
func (s *messageStore) listStarredMessages(ctx context.Context, chatJID string) ([]StoredMessage, error) {
	query := selectMessagesSQL + `
		WHERE m.chat_jid = ? AND m.starred = 1
		ORDER BY m.timestamp DESC`
	rows, err := s.db.QueryContext(ctx, query, chatJID)
	if err != nil {
		return nil, err
	}
	return scanStoredMessages(rows)
}

// getStats returns total counts of chats and messages stored in the database.
func (s *messageStore) getStats(ctx context.Context) (chats int, messages int, err error) {
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM whatsdown_chats`).Scan(&chats)
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM whatsdown_messages`).Scan(&messages)
	return chats, messages, nil
}

// listMessages returns up to limit messages of a chat sent before the given
// time (pass the zero time for the newest), oldest first.
func (s *messageStore) listMessages(ctx context.Context, chatJID string, before time.Time, limit int) ([]StoredMessage, error) {
	beforeTS := int64(1<<63 - 1)
	if !before.IsZero() {
		beforeTS = before.Unix()
	}
	query := `
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
		FROM (
			SELECT * FROM whatsdown_messages
			WHERE chat_jid = ? AND timestamp < ?
			ORDER BY timestamp DESC
			LIMIT ?
		) m
		LEFT JOIN whatsmeow_contacts cnt ON cnt.their_jid = m.sender_jid
		LEFT JOIN whatsmeow_lid_map lm ON (lm.lid || '@lid') = m.sender_jid
		LEFT JOIN whatsmeow_contacts cnt_lid ON cnt_lid.their_jid = (lm.pn || '@s.whatsapp.net')
		ORDER BY m.timestamp ASC`
	rows, err := s.db.QueryContext(ctx, query, chatJID, beforeTS, limit)
	if err != nil {
		return nil, err
	}
	return scanStoredMessages(rows)
}

// getMessage returns a single stored message by chat JID and message ID.
func (s *messageStore) getMessage(ctx context.Context, chatJID, id string) (*StoredMessage, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT chat_jid, id, sender_jid, sender_name, from_me, timestamp, kind, text, edited, raw, starred, 0
		FROM whatsdown_messages
		WHERE chat_jid = ? AND id = ?`, chatJID, id)
	var m StoredMessage
	var ts int64
	var fromMe, edited, starred, senderSaved int
	if err := row.Scan(&m.ChatJID, &m.ID, &m.SenderJID, &m.SenderName, &fromMe, &ts, &m.Kind, &m.Text, &edited, &m.Raw, &starred, &senderSaved); err != nil {
		return nil, err
	}
	m.FromMe = fromMe != 0
	m.Edited = edited != 0
	m.Starred = starred != 0
	m.SenderSaved = senderSaved != 0
	m.Timestamp = time.Unix(ts, 0)
	return &m, nil
}

// getMessageByID returns a single stored message by message ID across any chat.
func (s *messageStore) getMessageByID(ctx context.Context, id string) (*StoredMessage, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT chat_jid, id, sender_jid, sender_name, from_me, timestamp, kind, text, edited, raw, starred, 0
		FROM whatsdown_messages
		WHERE id = ? LIMIT 1`, id)
	var m StoredMessage
	var ts int64
	var fromMe, edited, starred, senderSaved int
	if err := row.Scan(&m.ChatJID, &m.ID, &m.SenderJID, &m.SenderName, &fromMe, &ts, &m.Kind, &m.Text, &edited, &m.Raw, &starred, &senderSaved); err != nil {
		return nil, err
	}
	m.FromMe = fromMe != 0
	m.Edited = edited != 0
	m.Starred = starred != 0
	m.SenderSaved = senderSaved != 0
	m.Timestamp = time.Unix(ts, 0)
	return &m, nil
}

// PollResults holds the aggregated results of a poll.
type PollResults struct {
	TotalVoters    int
	VoteCounts     map[string]int
	VotersByOption map[string][]string
}

// getPollResults returns aggregated vote counts and voters per option.
func (s *messageStore) getPollResults(ctx context.Context, chatJID, messageID string) (*PollResults, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT option_name, voter_jid FROM whatsdown_poll_votes WHERE chat_jid = ? AND message_id = ?`,
		chatJID, messageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := &PollResults{
		VoteCounts:     make(map[string]int),
		VotersByOption: make(map[string][]string),
	}
	voterSet := make(map[string]bool)

	for rows.Next() {
		var opt, voter string
		if err := rows.Scan(&opt, &voter); err == nil {
			results.VoteCounts[opt]++
			results.VotersByOption[opt] = append(results.VotersByOption[opt], voter)
			voterSet[voter] = true
		}
	}
	results.TotalVoters = len(voterSet)
	return results, nil
}

// getPollVotes returns a map of optionName -> true voted by the specified voter (defaults to 'me').
func (s *messageStore) getPollVotes(ctx context.Context, chatJID, messageID string, voterJID ...string) (map[string]bool, error) {
	voter := "me"
	if len(voterJID) > 0 && voterJID[0] != "" {
		voter = voterJID[0]
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT option_name FROM whatsdown_poll_votes WHERE chat_jid = ? AND message_id = ? AND (voter_jid = ? OR voter_jid = 'me')`,
		chatJID, messageID, voter)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res := make(map[string]bool)
	for rows.Next() {
		var opt string
		if err := rows.Scan(&opt); err == nil {
			res[opt] = true
		}
	}
	return res, nil
}

// setPollVotesForVoter replaces all votes for a specific voter on a poll message.
func (s *messageStore) setPollVotesForVoter(ctx context.Context, chatJID, messageID, voterJID string, options []string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if voterJID == "" {
		voterJID = "me"
	}

	_, err = tx.ExecContext(ctx,
		`DELETE FROM whatsdown_poll_votes WHERE chat_jid = ? AND message_id = ? AND voter_jid = ?`,
		chatJID, messageID, voterJID)
	if err != nil {
		return err
	}

	for _, opt := range options {
		_, err = tx.ExecContext(ctx,
			`INSERT INTO whatsdown_poll_votes (chat_jid, message_id, voter_jid, option_name) VALUES (?, ?, ?, ?)`,
			chatJID, messageID, voterJID, opt)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// togglePollVote toggles a vote on an option for a voter, respecting allowMultiple.
// It returns the updated set of selected options for that poll message.
func (s *messageStore) togglePollVote(ctx context.Context, chatJID, messageID, optionName string, allowMultiple bool, voterJID ...string) (map[string]bool, error) {
	voter := "me"
	if len(voterJID) > 0 && voterJID[0] != "" {
		voter = voterJID[0]
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if !allowMultiple {
		// Single choice: check if this option is already selected
		var count int
		_ = tx.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM whatsdown_poll_votes WHERE chat_jid = ? AND message_id = ? AND (voter_jid = ? OR voter_jid = 'me') AND option_name = ?`,
			chatJID, messageID, voter, optionName).Scan(&count)

		// Delete all previous votes by this voter
		_, err = tx.ExecContext(ctx,
			`DELETE FROM whatsdown_poll_votes WHERE chat_jid = ? AND message_id = ? AND (voter_jid = ? OR voter_jid = 'me')`,
			chatJID, messageID, voter)
		if err != nil {
			return nil, err
		}

		// If it wasn't selected before, select it now (toggle on)
		if count == 0 {
			_, err = tx.ExecContext(ctx,
				`INSERT INTO whatsdown_poll_votes (chat_jid, message_id, voter_jid, option_name) VALUES (?, ?, ?, ?)`,
				chatJID, messageID, voter, optionName)
			if err != nil {
				return nil, err
			}
		}
	} else {
		// Multi choice: toggle only this specific option
		var count int
		_ = tx.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM whatsdown_poll_votes WHERE chat_jid = ? AND message_id = ? AND (voter_jid = ? OR voter_jid = 'me') AND option_name = ?`,
			chatJID, messageID, voter, optionName).Scan(&count)

		if count > 0 {
			_, err = tx.ExecContext(ctx,
				`DELETE FROM whatsdown_poll_votes WHERE chat_jid = ? AND message_id = ? AND (voter_jid = ? OR voter_jid = 'me') AND option_name = ?`,
				chatJID, messageID, voter, optionName)
		} else {
			_, err = tx.ExecContext(ctx,
				`INSERT INTO whatsdown_poll_votes (chat_jid, message_id, voter_jid, option_name) VALUES (?, ?, ?, ?)`,
				chatJID, messageID, voter, optionName)
		}
		if err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.getPollVotes(ctx, chatJID, messageID, voter)
}
