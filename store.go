package main

// Chat and message storage.
//
// whatsmeow's sqlstore only keeps the login session (keys, contacts, app
// state). Chats and messages arrive as events and are persisted here, in our
// own whatsdown_* tables inside the same SQLite file.

import (
	"context"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/proto/waHistorySync"
	"go.mau.fi/whatsmeow/proto/waWeb"
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

	// 7: message status tracking (sent, delivered, read)
	`
	ALTER TABLE whatsdown_messages ADD COLUMN status INTEGER NOT NULL DEFAULT 0;
	CREATE INDEX IF NOT EXISTS whatsdown_messages_status ON whatsdown_messages (id, from_me);
	`,

	// 8: backfill outgoing status for existing synced messages
	`
	UPDATE whatsdown_messages SET status = 3 WHERE from_me = 1 AND status = 0;
	`,

	// 9: saved stickers support
	`
	CREATE TABLE IF NOT EXISTS whatsdown_saved_stickers (
		id          TEXT PRIMARY KEY,
		file_path   TEXT NOT NULL,
		file_hash   TEXT NOT NULL DEFAULT '',
		width       INTEGER NOT NULL DEFAULT 512,
		height      INTEGER NOT NULL DEFAULT 512,
		saved_at    INTEGER NOT NULL
	);
	CREATE INDEX IF NOT EXISTS whatsdown_saved_stickers_saved_at ON whatsdown_saved_stickers (saved_at DESC);
	`,

	// 10: message edit history
	`
	CREATE TABLE IF NOT EXISTS whatsdown_message_edits (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		message_id  TEXT NOT NULL,
		chat_jid    TEXT NOT NULL,
		text        TEXT NOT NULL,
		edited_at   INTEGER NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_whatsdown_message_edits_msg ON whatsdown_message_edits (chat_jid, message_id);
	`,

	// 11: group communities support
	`
	ALTER TABLE whatsdown_chats ADD COLUMN parent_jid TEXT NOT NULL DEFAULT '';
	ALTER TABLE whatsdown_chats ADD COLUMN is_parent INTEGER NOT NULL DEFAULT 0;
	ALTER TABLE whatsdown_chats ADD COLUMN is_default_sub_group INTEGER NOT NULL DEFAULT 0;
	CREATE INDEX IF NOT EXISTS idx_whatsdown_chats_parent ON whatsdown_chats (parent_jid);
	`,

	// 12: call history log
	`
	CREATE TABLE IF NOT EXISTS whatsdown_calls (
		id            TEXT PRIMARY KEY,
		chat_jid      TEXT NOT NULL,
		caller_jid    TEXT NOT NULL,
		caller_name   TEXT NOT NULL DEFAULT '',
		timestamp     INTEGER NOT NULL,
		duration_secs INTEGER NOT NULL DEFAULT 0,
		is_video      INTEGER NOT NULL DEFAULT 0,
		is_group      INTEGER NOT NULL DEFAULT 0,
		outcome       INTEGER NOT NULL DEFAULT 0,
		from_me       INTEGER NOT NULL DEFAULT 0
	);
	CREATE INDEX IF NOT EXISTS idx_whatsdown_calls_time ON whatsdown_calls (timestamp DESC);
	`,

	// 13: reactions support
	`
	CREATE TABLE IF NOT EXISTS whatsdown_reactions (
		chat_jid    TEXT NOT NULL,
		message_id  TEXT NOT NULL,
		sender_jid  TEXT NOT NULL,
		sender_name TEXT NOT NULL DEFAULT '',
		emoji       TEXT NOT NULL,
		from_me     INTEGER NOT NULL DEFAULT 0,
		timestamp   INTEGER NOT NULL DEFAULT 0,
		PRIMARY KEY (chat_jid, message_id, sender_jid)
	);
	CREATE INDEX IF NOT EXISTS idx_whatsdown_reactions_msg ON whatsdown_reactions (chat_jid, message_id);
	`,
}

// Delivery / read status of outgoing messages.
const (
	MessageStatusNone      = 0 // pending / unacknowledged
	MessageStatusSent      = 1 // server ack (single check)
	MessageStatusDelivered = 2 // delivered to recipient (double check)
	MessageStatusRead      = 3 // read / viewed by recipient (blue double check)
)

// Call outcome constants.
const (
	CallOutcomeConnected = 0
	CallOutcomeMissed    = 1
	CallOutcomeFailed    = 2
	CallOutcomeRejected  = 3
	CallOutcomeDeclined  = 4
)

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
	messageKindCall     = "call"
	messageKindDeleted  = "deleted"
)

// CallRecord represents a WhatsApp call history entry.
type CallRecord struct {
	ID           string
	ChatJID      string
	CallerJID    string
	CallerName   string
	Timestamp    time.Time
	DurationSecs int
	IsVideo      bool
	IsGroup      bool
	Outcome      int
	FromMe       bool
}

type messageStore struct {
	db            *sql.DB
	selfMu        sync.RWMutex
	selfPN        string
	selfLID       string
	stickerSyncMu sync.Mutex
}

// execer is satisfied by both *sql.DB and *sql.Tx.
type execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
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
			status := MessageStatusNone
			if evt.Info.IsFromMe {
				switch hsm.GetMessage().GetStatus() {
				case waWeb.WebMessageInfo_SERVER_ACK:
					status = MessageStatusSent
				case waWeb.WebMessageInfo_DELIVERY_ACK:
					status = MessageStatusDelivered
				case waWeb.WebMessageInfo_READ, waWeb.WebMessageInfo_PLAYED:
					status = MessageStatusRead
				default:
					status = MessageStatusSent
				}
			}
			inserted, err := s.saveMessageTxWithStar(ctx, tx, evt, isStarred, status)
			if err != nil {
				return 0, err
			}
			if inserted {
				stored++
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	favs := data.GetFavoriteStickers()
	recents := data.GetRecentStickers()
	if (len(favs) > 0 || len(recents) > 0) && client != nil {
		go s.syncHistoryStickers(context.Background(), client, favs, recents)
	}
	if client != nil {
		go s.importStickersFromChatHistory(context.Background(), client)
	}

	return stored, nil
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
func (s *messageStore) saveMessageTxWithStar(ctx context.Context, tx execer, evt *events.Message, starred bool, initialStatus ...int) (bool, error) {
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
			_, _ = tx.ExecContext(ctx, `DELETE FROM whatsdown_reactions WHERE chat_jid = ? AND message_id = ?`, chat, targetID)
			return false, err
		case waE2E.ProtocolMessage_MESSAGE_EDIT:
			if _, text, ok := describeMessage(pm.GetEditedMessage()); ok {
				var prevText string
				var prevTs int64
				if err := tx.QueryRowContext(ctx, `SELECT text, timestamp FROM whatsdown_messages WHERE chat_jid = ? AND id = ?`, chat, targetID).Scan(&prevText, &prevTs); err == nil && prevText != "" {
					_, _ = tx.ExecContext(ctx, `INSERT INTO whatsdown_message_edits (message_id, chat_jid, text, edited_at) VALUES (?, ?, ?, ?)`, targetID, chat, prevText, prevTs)
				}
				raw, _ := proto.Marshal(pm.GetEditedMessage())
				_, err := tx.ExecContext(ctx,
					`UPDATE whatsdown_messages SET text = ?, raw = ?, edited = 1 WHERE chat_jid = ? AND id = ?`,
					text, raw, chat, targetID)
				return false, err
			}
		}
		return false, nil
	}

	var rm *waE2E.ReactionMessage
	if evt.Message != nil {
		rm = evt.Message.GetReactionMessage()
	}
	if rm != nil && rm.GetKey() != nil {
		targetID := rm.GetKey().GetID()
		if targetID != "" {
			emoji := rm.GetText()
			senderJID := evt.Info.Sender.ToNonAD().String()
			if senderJID == "" {
				senderJID = chat
			}
			senderName := evt.Info.PushName
			fromMeVal := 0
			if evt.Info.IsFromMe {
				fromMeVal = 1
			}
			ts := evt.Info.Timestamp.Unix()
			if ts <= 0 {
				ts = time.Now().Unix()
			}
			if emoji == "" {
				_, _ = tx.ExecContext(ctx, `DELETE FROM whatsdown_reactions WHERE chat_jid = ? AND message_id = ? AND sender_jid = ?`, chat, targetID, senderJID)
			} else {
				_, _ = tx.ExecContext(ctx, `
					INSERT INTO whatsdown_reactions (chat_jid, message_id, sender_jid, sender_name, emoji, from_me, timestamp)
					VALUES (?, ?, ?, ?, ?, ?, ?)
					ON CONFLICT(chat_jid, message_id, sender_jid)
					DO UPDATE SET emoji = excluded.emoji, sender_name = excluded.sender_name, from_me = excluded.from_me, timestamp = excluded.timestamp`,
					chat, targetID, senderJID, senderName, emoji, fromMeVal, ts)
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

	if evt.Message != nil && evt.Message.GetCallLogMesssage() != nil {
		cl := evt.Message.GetCallLogMesssage()
		callerJID := evt.Info.Sender.ToNonAD().String()
		if callerJID == "" {
			callerJID = chat
		}
		outcome := CallOutcomeConnected
		switch cl.GetCallOutcome() {
		case waE2E.CallLogMessage_CONNECTED:
			outcome = CallOutcomeConnected
		case waE2E.CallLogMessage_MISSED:
			outcome = CallOutcomeMissed
		case waE2E.CallLogMessage_FAILED:
			outcome = CallOutcomeFailed
		case waE2E.CallLogMessage_REJECTED:
			outcome = CallOutcomeRejected
		case waE2E.CallLogMessage_ACCEPTED_ELSEWHERE, waE2E.CallLogMessage_SILENCED_BY_DND, waE2E.CallLogMessage_SILENCED_UNKNOWN_CALLER:
			outcome = CallOutcomeDeclined
		}
		_ = s.saveCallTx(ctx, tx, CallRecord{
			ID:           evt.Info.ID,
			ChatJID:      chat,
			CallerJID:    callerJID,
			CallerName:   evt.Info.PushName,
			Timestamp:    evt.Info.Timestamp,
			DurationSecs: int(cl.GetDurationSecs()),
			IsVideo:      cl.GetIsVideo(),
			IsGroup:      evt.Info.IsGroup,
			Outcome:      outcome,
			FromMe:       evt.Info.IsFromMe,
		})
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

	statusVal := MessageStatusNone
	if len(initialStatus) > 0 && initialStatus[0] > 0 {
		statusVal = initialStatus[0]
	} else if isSelf {
		statusVal = MessageStatusRead
	} else if fromMeVal == 1 {
		statusVal = MessageStatusSent
	}

	res, err := tx.ExecContext(ctx, `
		INSERT INTO whatsdown_messages (chat_jid, id, sender_jid, sender_name, from_me, timestamp, kind, text, raw, starred, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (chat_jid, id) DO UPDATE SET
			starred = CASE WHEN excluded.starred = 1 THEN 1 ELSE whatsdown_messages.starred END,
			from_me = CASE WHEN excluded.from_me = 1 THEN 1 ELSE whatsdown_messages.from_me END,
			status  = MAX(whatsdown_messages.status, excluded.status)`,
		chat, evt.Info.ID, evt.Info.Sender.ToNonAD().String(), evt.Info.PushName,
		fromMeVal, ts, kind, text, raw, starredInt, statusVal)
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
	case m.GetCallLogMesssage() != nil:
		cl := m.GetCallLogMesssage()
		text := "Voice call"
		if cl.GetIsVideo() {
			text = "Video call"
		}
		if cl.GetCallOutcome() == waE2E.CallLogMessage_MISSED {
			text = "Missed " + strings.ToLower(text)
		} else if cl.GetDurationSecs() > 0 {
			text = fmt.Sprintf("%s (%s)", text, formatCallDuration(int(cl.GetDurationSecs())))
		}
		return messageKindCall, text, true
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

func formatCallDuration(secs int) string {
	if secs <= 0 {
		return "0s"
	}
	m := secs / 60
	s := secs % 60
	if m == 0 {
		return fmt.Sprintf("%ds", s)
	}
	h := m / 60
	m = m % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	}
	return fmt.Sprintf("%dm %ds", m, s)
}

// Chat is a row of whatsdown_chats with the latest message preview.
type Chat struct {
	JID               string
	Name              string
	LastMessageAt     time.Time
	UnreadCount       int
	Archived          bool
	Pinned            bool
	Muted             bool
	ParentJID         string
	IsParent          bool
	IsDefaultSubGroup bool
	LastMessageText   string
	LastMessageSender string
	LastMessageFromMe bool
	LastMessageKind   string
	LastMessageStatus int
}

// Reaction represents an emoji reaction on a message.
type Reaction struct {
	ChatJID    string    `json:"chat_jid"`
	MessageID  string    `json:"message_id"`
	SenderJID  string    `json:"sender_jid"`
	SenderName string    `json:"sender_name"`
	Emoji      string    `json:"emoji"`
	FromMe     bool      `json:"from_me"`
	Timestamp  time.Time `json:"timestamp"`
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
	Status      int
	SenderSaved bool
	Reactions   []Reaction
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
			c.muted,
			COALESCE(c.parent_jid, ''),
			COALESCE(c.is_parent, 0),
			COALESCE(c.is_default_sub_group, 0),
			COALESCE(lm_msg.text, ''),
			COALESCE(
				NULLIF(lm_cnt.full_name, ''),
				NULLIF(lm_cnt_lid.full_name, ''),
				NULLIF(lm_cnt.first_name, ''),
				NULLIF(lm_cnt_lid.first_name, ''),
				NULLIF(lm_cnt.business_name, ''),
				NULLIF(lm_cnt_lid.business_name, ''),
				NULLIF(lm_cnt.push_name, ''),
				NULLIF(lm_cnt_lid.push_name, ''),
				NULLIF(lm_msg.sender_name, ''),
				CASE 
					WHEN lm_msg.sender_jid LIKE '%@s.whatsapp.net' THEN '+' || SUBSTR(lm_msg.sender_jid, 1, INSTR(lm_msg.sender_jid, '@') - 1)
					WHEN lm_lid.pn IS NOT NULL AND lm_lid.pn <> '' THEN '+' || lm_lid.pn
					ELSE ''
				END
			) AS resolved_last_sender_name,
			COALESCE(lm_msg.from_me, 0),
			COALESCE(lm_msg.kind, ''),
			COALESCE(lm_msg.status, 0)
		FROM whatsdown_chats c
		LEFT JOIN whatsmeow_contacts cnt ON cnt.their_jid = c.jid
		LEFT JOIN whatsmeow_lid_map lm ON (lm.lid || '@lid') = c.jid
		LEFT JOIN whatsmeow_contacts cnt_lid ON cnt_lid.their_jid = (lm.pn || '@s.whatsapp.net')
		LEFT JOIN whatsdown_messages lm_msg ON lm_msg.rowid = (
			SELECT m2.rowid
			FROM whatsdown_messages m2
			WHERE m2.chat_jid = c.jid
			ORDER BY m2.timestamp DESC, m2.rowid DESC
			LIMIT 1
		)
		LEFT JOIN whatsmeow_contacts lm_cnt ON lm_cnt.their_jid = lm_msg.sender_jid
		LEFT JOIN whatsmeow_lid_map lm_lid ON (lm_lid.lid || '@lid') = lm_msg.sender_jid
		LEFT JOIN whatsmeow_contacts lm_cnt_lid ON lm_cnt_lid.their_jid = (lm_lid.pn || '@s.whatsapp.net')
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
		var fromMe, isParent, isDefaultSub int
		if err := rows.Scan(
			&c.JID,
			&c.Name,
			&ts,
			&c.UnreadCount,
			&c.Archived,
			&c.Pinned,
			&c.Muted,
			&c.ParentJID,
			&isParent,
			&isDefaultSub,
			&c.LastMessageText,
			&c.LastMessageSender,
			&fromMe,
			&c.LastMessageKind,
			&c.LastMessageStatus,
		); err != nil {
			return nil, err
		}
		c.IsParent = isParent != 0
		c.IsDefaultSubGroup = isDefaultSub != 0
		c.LastMessageFromMe = fromMe != 0
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

// getChatOrContact retrieves a Chat by JID, resolving contact names, metadata, and latest message preview.
func (s *messageStore) getChatOrContact(ctx context.Context, jid string) (Chat, error) {
	var name string
	var lastMsgAt int64
	var unread int
	var archived, pinned, muted bool
	var parentJID string
	var isParent, isDefaultSub int
	var lastMsgText, lastMsgSender, lastMsgKind string
	var fromMe, lastMsgStatus int

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
			COALESCE(c.muted, 0),
			COALESCE(c.parent_jid, ''),
			COALESCE(c.is_parent, 0),
			COALESCE(c.is_default_sub_group, 0),
			COALESCE(lm_msg.text, ''),
			COALESCE(
				NULLIF(lm_cnt.full_name, ''),
				NULLIF(lm_cnt_lid.full_name, ''),
				NULLIF(lm_cnt.first_name, ''),
				NULLIF(lm_cnt_lid.first_name, ''),
				NULLIF(lm_cnt.business_name, ''),
				NULLIF(lm_cnt_lid.business_name, ''),
				NULLIF(lm_cnt.push_name, ''),
				NULLIF(lm_cnt_lid.push_name, ''),
				NULLIF(lm_msg.sender_name, ''),
				CASE 
					WHEN lm_msg.sender_jid LIKE '%@s.whatsapp.net' THEN '+' || SUBSTR(lm_msg.sender_jid, 1, INSTR(lm_msg.sender_jid, '@') - 1)
					WHEN lm_lid.pn IS NOT NULL AND lm_lid.pn <> '' THEN '+' || lm_lid.pn
					ELSE ''
				END
			) AS resolved_last_sender_name,
			COALESCE(lm_msg.from_me, 0),
			COALESCE(lm_msg.kind, ''),
			COALESCE(lm_msg.status, 0)
		FROM (SELECT ? AS jid) t
		LEFT JOIN whatsdown_chats c ON c.jid = t.jid
		LEFT JOIN whatsmeow_contacts cnt ON cnt.their_jid = t.jid
		LEFT JOIN whatsmeow_lid_map lm ON (lm.lid || '@lid') = t.jid
		LEFT JOIN whatsmeow_contacts cnt_lid ON cnt_lid.their_jid = (lm.pn || '@s.whatsapp.net')
		LEFT JOIN whatsdown_messages lm_msg ON lm_msg.rowid = (
			SELECT m2.rowid
			FROM whatsdown_messages m2
			WHERE m2.chat_jid = t.jid
			ORDER BY m2.timestamp DESC, m2.rowid DESC
			LIMIT 1
		)
		LEFT JOIN whatsmeow_contacts lm_cnt ON lm_cnt.their_jid = lm_msg.sender_jid
		LEFT JOIN whatsmeow_lid_map lm_lid ON (lm_lid.lid || '@lid') = lm_msg.sender_jid
		LEFT JOIN whatsmeow_contacts lm_cnt_lid ON lm_cnt_lid.their_jid = (lm_lid.pn || '@s.whatsapp.net')`,
		jid).Scan(
		&name, &lastMsgAt, &unread, &archived, &pinned, &muted,
		&parentJID, &isParent, &isDefaultSub,
		&lastMsgText, &lastMsgSender, &fromMe, &lastMsgKind, &lastMsgStatus,
	)
	if err != nil {
		return Chat{JID: jid}, err
	}
	return Chat{
		JID:               jid,
		Name:              name,
		LastMessageAt:     time.Unix(lastMsgAt, 0),
		UnreadCount:       unread,
		Archived:          archived,
		Pinned:            pinned,
		Muted:             muted,
		ParentJID:         parentJID,
		IsParent:          isParent != 0,
		IsDefaultSubGroup: isDefaultSub != 0,
		LastMessageText:   lastMsgText,
		LastMessageSender: lastMsgSender,
		LastMessageFromMe: fromMe != 0,
		LastMessageKind:   lastMsgKind,
		LastMessageStatus: lastMsgStatus,
	}, nil
}

// setChatCommunity sets community hierarchy information for a chat.
func (s *messageStore) setChatCommunity(ctx context.Context, jid string, parentJID string, isParent bool, isDefaultSubGroup bool) error {
	isP := 0
	if isParent {
		isP = 1
	}
	isD := 0
	if isDefaultSubGroup {
		isD = 1
	}
	_, err := s.db.ExecContext(ctx, `
		UPDATE whatsdown_chats
		SET parent_jid = ?, is_parent = ?, is_default_sub_group = ?
		WHERE jid = ?`, parentJID, isP, isD, jid)
	return err
}

// syncGroups updates community and parent-child group relations from whatsmeow.
func (s *messageStore) syncGroups(ctx context.Context, groups []*types.GroupInfo) error {
	if len(groups) == 0 {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, g := range groups {
		if g == nil {
			continue
		}
		jid := g.JID.ToNonAD().String()
		name := g.GroupName.Name
		isParent := 0
		if g.GroupParent.IsParent {
			isParent = 1
		}
		parentJID := ""
		if !g.GroupLinkedParent.LinkedParentJID.IsEmpty() {
			parentJID = g.GroupLinkedParent.LinkedParentJID.ToNonAD().String()
		}
		isDefaultSub := 0
		if g.GroupIsDefaultSub.IsDefaultSubGroup {
			isDefaultSub = 1
		}

		if parentJID != "" {
			_, _ = tx.ExecContext(ctx, `
				INSERT INTO whatsdown_chats (jid, name, is_parent)
				VALUES (?, 'Community', 1)
				ON CONFLICT (jid) DO UPDATE SET is_parent = 1`,
				parentJID)
		}

		_, err = tx.ExecContext(ctx, `
			INSERT INTO whatsdown_chats (jid, name, parent_jid, is_parent, is_default_sub_group)
			VALUES (?, ?, ?, ?, ?)
			ON CONFLICT (jid) DO UPDATE SET
				name = CASE WHEN excluded.name <> '' THEN excluded.name ELSE name END,
				parent_jid = excluded.parent_jid,
				is_parent = excluded.is_parent,
				is_default_sub_group = excluded.is_default_sub_group`,
			jid, name, parentJID, isParent, isDefaultSub)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *messageStore) saveCallTx(ctx context.Context, tx execer, call CallRecord) error {
	isVideoVal := 0
	if call.IsVideo {
		isVideoVal = 1
	}
	isGroupVal := 0
	if call.IsGroup {
		isGroupVal = 1
	}
	fromMeVal := 0
	if call.FromMe {
		fromMeVal = 1
	}
	ts := call.Timestamp.Unix()
	if ts <= 0 {
		ts = time.Now().Unix()
	}

	_, err := tx.ExecContext(ctx, `
		INSERT INTO whatsdown_calls (
			id, chat_jid, caller_jid, caller_name, timestamp,
			duration_secs, is_video, is_group, outcome, from_me
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (id) DO UPDATE SET
			caller_name   = CASE WHEN excluded.caller_name <> '' THEN excluded.caller_name ELSE caller_name END,
			duration_secs = CASE WHEN excluded.duration_secs > 0 THEN excluded.duration_secs ELSE duration_secs END,
			outcome       = excluded.outcome`,
		call.ID, call.ChatJID, call.CallerJID, call.CallerName, ts,
		call.DurationSecs, isVideoVal, isGroupVal, call.Outcome, fromMeVal)
	return err
}

func (s *messageStore) saveCall(ctx context.Context, call CallRecord) error {
	return s.saveCallTx(ctx, s.db, call)
}

func (s *messageStore) listCalls(ctx context.Context, query string) ([]CallRecord, error) {
	sqlQuery := `
		SELECT 
			c.id,
			c.chat_jid,
			c.caller_jid,
			COALESCE(
				NULLIF(cnt.full_name, ''),
				NULLIF(cnt_lid.full_name, ''),
				NULLIF(cnt.first_name, ''),
				NULLIF(cnt_lid.first_name, ''),
				NULLIF(cnt.business_name, ''),
				NULLIF(cnt_lid.business_name, ''),
				NULLIF(cnt.push_name, ''),
				NULLIF(cnt_lid.push_name, ''),
				NULLIF(ch.name, ''),
				NULLIF(c.caller_name, ''),
				CASE WHEN lm.pn IS NOT NULL AND lm.pn <> '' THEN '+' || lm.pn ELSE NULL END,
				c.caller_jid
			) AS resolved_caller_name,
			c.timestamp,
			c.duration_secs,
			c.is_video,
			c.is_group,
			c.outcome,
			c.from_me
		FROM whatsdown_calls c
		LEFT JOIN whatsmeow_contacts cnt ON cnt.their_jid = c.caller_jid
		LEFT JOIN whatsmeow_lid_map lm ON (lm.lid || '@lid') = c.caller_jid
		LEFT JOIN whatsmeow_contacts cnt_lid ON cnt_lid.their_jid = (lm.pn || '@s.whatsapp.net')
		LEFT JOIN whatsdown_chats ch ON ch.jid = c.chat_jid
	`
	var args []any
	if query != "" {
		sqlQuery += ` WHERE (ch.name LIKE ? OR cnt.full_name LIKE ? OR cnt.push_name LIKE ? OR c.caller_name LIKE ? OR c.caller_jid LIKE ?)`
		like := "%" + query + "%"
		args = append(args, like, like, like, like, like)
	}
	sqlQuery += ` ORDER BY c.timestamp DESC LIMIT 200`

	rows, err := s.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var calls []CallRecord
	for rows.Next() {
		var rec CallRecord
		var ts int64
		var isVideo, isGroup, fromMe int
		if err := rows.Scan(
			&rec.ID,
			&rec.ChatJID,
			&rec.CallerJID,
			&rec.CallerName,
			&ts,
			&rec.DurationSecs,
			&isVideo,
			&isGroup,
			&rec.Outcome,
			&fromMe,
		); err != nil {
			return nil, err
		}
		rec.Timestamp = time.Unix(ts, 0)
		rec.IsVideo = isVideo != 0
		rec.IsGroup = isGroup != 0
		rec.FromMe = fromMe != 0
		calls = append(calls, rec)
	}
	return calls, rows.Err()
}

func (s *messageStore) deleteCall(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM whatsdown_calls WHERE id = ?`, id)
	return err
}

func (s *messageStore) clearCalls(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM whatsdown_calls`)
	return err
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

// resolveLIDtoPN maps a LID JID to a phone number JID if present in whatsmeow_lid_map.
func (s *messageStore) resolveLIDtoPN(ctx context.Context, jid string) string {
	if !strings.HasSuffix(jid, "@lid") {
		return jid
	}
	clean := strings.TrimSuffix(jid, "@lid")
	clean = strings.Split(clean, ":")[0]
	var pn string
	err := s.db.QueryRowContext(ctx, `SELECT pn FROM whatsmeow_lid_map WHERE lid = ? LIMIT 1`, clean).Scan(&pn)
	if err == nil && pn != "" {
		return pn + "@s.whatsapp.net"
	}
	return jid
}

// toggleMessageStarred updates the starred status of a message, handling both LID and phone number JIDs.
func (s *messageStore) toggleMessageStarred(ctx context.Context, chatJID, id string, starred bool) error {
	starredInt := 0
	if starred {
		starredInt = 1
	}
	_, err := s.db.ExecContext(ctx, `
		UPDATE whatsdown_messages SET starred = ?
		WHERE id = ? AND (
			chat_jid = ?
			OR chat_jid IN (SELECT lid || '@lid' FROM whatsmeow_lid_map WHERE (pn || '@s.whatsapp.net') = ?)
			OR chat_jid IN (SELECT pn || '@s.whatsapp.net' FROM whatsmeow_lid_map WHERE (lid || '@lid') = ?)
		)`, starredInt, id, chatJID, chatJID, chatJID)
	return err
}

func scanStoredMessages(rows *sql.Rows) ([]StoredMessage, error) {
	defer rows.Close()
	var msgs []StoredMessage
	for rows.Next() {
		var m StoredMessage
		var ts int64
		if err := rows.Scan(&m.ChatJID, &m.ID, &m.SenderJID, &m.SenderName, &m.FromMe, &ts, &m.Kind, &m.Text, &m.Edited, &m.Raw, &m.Starred, &m.Status, &m.SenderSaved); err != nil {
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
	msgs, err := scanStoredMessages(rows)
	if err != nil {
		return nil, err
	}
	if len(msgs) > 0 {
		if rMap, err := s.getReactionsForChat(ctx, chatJID); err == nil {
			for i := range msgs {
				if rList, ok := rMap[msgs[i].ID]; ok {
					msgs[i].Reactions = rList
				}
			}
		}
	}
	return msgs, nil
}

// getMessage returns a single stored message by chat JID and message ID.
func (s *messageStore) getMessage(ctx context.Context, chatJID, id string) (*StoredMessage, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT chat_jid, id, sender_jid, sender_name, from_me, timestamp, kind, text, edited, raw, starred, status, 0
		FROM whatsdown_messages
		WHERE chat_jid = ? AND id = ?`, chatJID, id)
	var m StoredMessage
	var ts int64
	var fromMe, edited, starred, senderSaved int
	if err := row.Scan(&m.ChatJID, &m.ID, &m.SenderJID, &m.SenderName, &fromMe, &ts, &m.Kind, &m.Text, &edited, &m.Raw, &starred, &m.Status, &senderSaved); err != nil {
		return nil, err
	}
	m.FromMe = fromMe != 0
	m.Edited = edited != 0
	m.Starred = starred != 0
	m.SenderSaved = senderSaved != 0
	m.Timestamp = time.Unix(ts, 0)
	if rList, err := s.getReactionsForMessage(ctx, m.ChatJID, m.ID); err == nil {
		m.Reactions = rList
	}
	return &m, nil
}

// getMessageByID returns a single stored message by message ID across any chat.
func (s *messageStore) getMessageByID(ctx context.Context, id string) (*StoredMessage, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT chat_jid, id, sender_jid, sender_name, from_me, timestamp, kind, text, edited, raw, starred, status, 0
		FROM whatsdown_messages
		WHERE id = ? LIMIT 1`, id)
	var m StoredMessage
	var ts int64
	var fromMe, edited, starred, senderSaved int
	if err := row.Scan(&m.ChatJID, &m.ID, &m.SenderJID, &m.SenderName, &fromMe, &ts, &m.Kind, &m.Text, &edited, &m.Raw, &starred, &m.Status, &senderSaved); err != nil {
		return nil, err
	}
	m.FromMe = fromMe != 0
	m.Edited = edited != 0
	m.Starred = starred != 0
	m.SenderSaved = senderSaved != 0
	m.Timestamp = time.Unix(ts, 0)
	if rList, err := s.getReactionsForMessage(ctx, m.ChatJID, m.ID); err == nil {
		m.Reactions = rList
	}
	return &m, nil
}

// getReactionsForChat loads all reactions for a given chat, grouped by message ID.
func (s *messageStore) getReactionsForChat(ctx context.Context, chatJID string) (map[string][]Reaction, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT chat_jid, message_id, sender_jid, sender_name, emoji, from_me, timestamp
		FROM whatsdown_reactions
		WHERE chat_jid = ?
		ORDER BY timestamp ASC`, chatJID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res := make(map[string][]Reaction)
	for rows.Next() {
		var r Reaction
		var ts int64
		var fromMeInt int
		if err := rows.Scan(&r.ChatJID, &r.MessageID, &r.SenderJID, &r.SenderName, &r.Emoji, &fromMeInt, &ts); err != nil {
			return nil, err
		}
		r.FromMe = fromMeInt != 0
		r.Timestamp = time.Unix(ts, 0)
		res[r.MessageID] = append(res[r.MessageID], r)
	}
	return res, rows.Err()
}

// getReactionsForMessage loads all reactions for a specific message.
func (s *messageStore) getReactionsForMessage(ctx context.Context, chatJID, messageID string) ([]Reaction, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT chat_jid, message_id, sender_jid, sender_name, emoji, from_me, timestamp
		FROM whatsdown_reactions
		WHERE chat_jid = ? AND message_id = ?
		ORDER BY timestamp ASC`, chatJID, messageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reactions []Reaction
	for rows.Next() {
		var r Reaction
		var ts int64
		var fromMeInt int
		if err := rows.Scan(&r.ChatJID, &r.MessageID, &r.SenderJID, &r.SenderName, &r.Emoji, &fromMeInt, &ts); err != nil {
			return nil, err
		}
		r.FromMe = fromMeInt != 0
		r.Timestamp = time.Unix(ts, 0)
		reactions = append(reactions, r)
	}
	return reactions, rows.Err()
}

// saveReaction inserts or updates a reaction on a message.
func (s *messageStore) saveReaction(ctx context.Context, r Reaction) error {
	fromMeInt := 0
	if r.FromMe {
		fromMeInt = 1
	}
	ts := r.Timestamp.Unix()
	if ts <= 0 {
		ts = time.Now().Unix()
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO whatsdown_reactions (chat_jid, message_id, sender_jid, sender_name, emoji, from_me, timestamp)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(chat_jid, message_id, sender_jid)
		DO UPDATE SET emoji = excluded.emoji, sender_name = excluded.sender_name, from_me = excluded.from_me, timestamp = excluded.timestamp`,
		r.ChatJID, r.MessageID, r.SenderJID, r.SenderName, r.Emoji, fromMeInt, ts)
	return err
}

// removeReaction deletes a reaction from a message by sender.
func (s *messageStore) removeReaction(ctx context.Context, chatJID, messageID, senderJID string) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM whatsdown_reactions
		WHERE chat_jid = ? AND message_id = ? AND sender_jid = ?`,
		chatJID, messageID, senderJID)
	return err
}

// removeMyReaction deletes the current user's reaction from a message.
func (s *messageStore) removeMyReaction(ctx context.Context, chatJID, messageID string) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM whatsdown_reactions
		WHERE chat_jid = ? AND message_id = ? AND from_me = 1`,
		chatJID, messageID)
	return err
}

// updateMessageStatuses updates the delivery/read status of outgoing messages.
func (s *messageStore) updateMessageStatuses(ctx context.Context, ids []string, status int) error {
	if len(ids) == 0 || status <= 0 {
		return nil
	}
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids)+1)
	args[0] = status
	for i, id := range ids {
		placeholders[i] = "?"
		args[i+1] = id
	}
	query := fmt.Sprintf(`UPDATE whatsdown_messages SET status = MAX(status, ?) WHERE id IN (%s)`,
		strings.Join(placeholders, ","))
	_, err := s.db.ExecContext(ctx, query, args...)
	return err
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

// SavedSticker represents a sticker stored in the user's personal sticker collection.
type SavedSticker struct {
	ID       string
	FilePath string
	FileHash string
	Width    int
	Height   int
	SavedAt  time.Time
}

// saveSticker adds or updates a sticker in the user's saved sticker collection.
func (s *messageStore) saveSticker(ctx context.Context, sticker SavedSticker) error {
	savedAt := sticker.SavedAt.Unix()
	if savedAt == 0 {
		savedAt = time.Now().Unix()
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO whatsdown_saved_stickers (id, file_path, file_hash, width, height, saved_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET file_path = excluded.file_path, file_hash = excluded.file_hash, saved_at = excluded.saved_at`,
		sticker.ID, sticker.FilePath, sticker.FileHash, sticker.Width, sticker.Height, savedAt)
	return err
}

// deleteSavedSticker removes a sticker from the user's saved stickers.
func (s *messageStore) deleteSavedSticker(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM whatsdown_saved_stickers WHERE id = ? OR file_hash = ?`, id, id)
	return err
}

// listSavedStickers returns all saved stickers ordered by newest first.
func (s *messageStore) listSavedStickers(ctx context.Context) ([]SavedSticker, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, file_path, file_hash, width, height, saved_at
		FROM whatsdown_saved_stickers
		ORDER BY saved_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SavedSticker
	for rows.Next() {
		var st SavedSticker
		var unixSaved int64
		if err := rows.Scan(&st.ID, &st.FilePath, &st.FileHash, &st.Width, &st.Height, &unixSaved); err != nil {
			return nil, err
		}
		st.SavedAt = time.Unix(unixSaved, 0)
		out = append(out, st)
	}
	return out, rows.Err()
}

// isStickerSaved checks if a sticker is in the saved collection by ID or SHA256 hash.
func (s *messageStore) isStickerSaved(ctx context.Context, idOrHash string) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM whatsdown_saved_stickers WHERE id = ? OR file_hash = ?`, idOrHash, idOrHash).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// syncHistoryStickers downloads favorite and recent stickers received in history sync chunks
// and stores them into the personal saved stickers collection.
func (s *messageStore) syncHistoryStickers(ctx context.Context, client *whatsmeow.Client, favorites, recents []*waHistorySync.StickerMetadata) {
	if client == nil {
		client = waClient
	}
	if client == nil {
		return
	}
	s.stickerSyncMu.Lock()
	defer s.stickerSyncMu.Unlock()

	targetDir := savedStickersDir()
	imported := 0

	processList := func(list []*waHistorySync.StickerMetadata, isFav bool) {
		for _, st := range list {
			if st == nil {
				continue
			}
			sha := st.GetFileSHA256()
			if len(sha) == 0 {
				continue
			}
			fileHash := hex.EncodeToString(sha)
			saved, _ := s.isStickerSaved(ctx, fileHash)
			if saved {
				continue
			}

			destPath := filepath.Join(targetDir, fileHash+".webp")
			var data []byte

			if fi, err := os.Stat(destPath); err == nil && fi.Size() > 0 {
				data, _ = os.ReadFile(destPath)
			}

			if len(data) == 0 {
				dlCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
				d, err := client.Download(dlCtx, st)
				cancel()
				if err != nil || len(d) == 0 {
					continue
				}
				data = d
				if err := os.WriteFile(destPath, data, 0o600); err != nil {
					continue
				}
			}

			w := int(st.GetWidth())
			h := int(st.GetHeight())
			if w <= 0 {
				w = 512
			}
			if h <= 0 {
				h = 512
			}

			if err := s.saveSticker(ctx, SavedSticker{
				ID:       fileHash,
				FileHash: fileHash,
				FilePath: destPath,
				Width:    w,
				Height:   h,
				SavedAt:  time.Now(),
			}); err == nil {
				imported++
				if imported%10 == 0 {
					glib.IdleAdd(func() {
						if activeMainWindow != nil {
							activeMainWindow.loadSavedStickersUI()
						}
					})
				}
			}
		}
	}

	processList(favorites, true)
	processList(recents, false)

	if imported > 0 {
		log.Printf("syncHistoryStickers: synced %d stickers from device", imported)
		glib.IdleAdd(func() {
			if activeMainWindow != nil {
				activeMainWindow.loadSavedStickersUI()
			}
		})
	}
}

// importStickersFromChatHistory checks the local database for sticker messages
// and imports/downloads any that are not yet in the saved stickers collection.
func (s *messageStore) importStickersFromChatHistory(ctx context.Context, client *whatsmeow.Client) int {
	if s.db == nil {
		return 0
	}
	s.stickerSyncMu.Lock()
	defer s.stickerSyncMu.Unlock()

	if client == nil {
		client = waClient
	}

	type stickerRow struct {
		id      string
		chatJID string
		raw     []byte
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, chat_jid, raw
		FROM whatsdown_messages
		WHERE kind = 'sticker' AND raw IS NOT NULL
		ORDER BY from_me DESC, timestamp DESC
	`)
	if err != nil {
		log.Printf("importStickersFromChatHistory query error: %v", err)
		return 0
	}
	defer rows.Close()

	var msgs []stickerRow
	for rows.Next() {
		var m stickerRow
		if err := rows.Scan(&m.id, &m.chatJID, &m.raw); err == nil {
			msgs = append(msgs, m)
		}
	}
	_ = rows.Close()

	imported := 0
	targetDir := savedStickersDir()
	seenHashes := make(map[string]bool)

	for _, m := range msgs {
		if ctx.Err() != nil {
			break
		}
		waMsg := parseE2EMessage(m.raw)
		if waMsg == nil || waMsg.GetStickerMessage() == nil {
			continue
		}
		stMsg := waMsg.GetStickerMessage()
		sha := stMsg.GetFileSHA256()
		if len(sha) == 0 {
			continue
		}
		hash := hex.EncodeToString(sha)
		if seenHashes[hash] {
			continue
		}
		seenHashes[hash] = true

		if saved, _ := s.isStickerSaved(ctx, hash); saved {
			continue
		}

		destPath := filepath.Join(targetDir, hash+".webp")
		var data []byte

		// 1. Check if destination already has the sticker file
		if fi, err := os.Stat(destPath); err == nil && fi.Size() > 0 {
			data, _ = os.ReadFile(destPath)
		}

		// 2. Check if media cache has the sticker
		if len(data) == 0 {
			localPath := getLocalMediaPath(StoredMessage{ID: m.id, ChatJID: m.chatJID, Raw: m.raw})
			if fi, err := os.Stat(localPath); err == nil && fi.Size() > 0 {
				if d, err := os.ReadFile(localPath); err == nil && len(d) > 0 {
					data = d
					_ = os.WriteFile(destPath, data, 0o600)
				}
			}
		}

		// 3. Download from WhatsApp MMS servers
		if len(data) == 0 && client != nil {
			dlCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
			d, err := client.Download(dlCtx, stMsg)
			cancel()
			if err == nil && len(d) > 0 {
				data = d
				_ = os.WriteFile(destPath, data, 0o600)
				localPath := getLocalMediaPath(StoredMessage{ID: m.id, ChatJID: m.chatJID, Raw: m.raw})
				_ = os.WriteFile(localPath, data, 0o600)
			} else if err != nil {
				log.Printf("importStickersFromChatHistory: download sticker %s failed: %v", hash, err)
			}
		}

		if len(data) == 0 {
			continue
		}

		w := int(stMsg.GetWidth())
		h := int(stMsg.GetHeight())
		if w <= 0 {
			w = 512
		}
		if h <= 0 {
			h = 512
		}

		if err := s.saveSticker(ctx, SavedSticker{
			ID:       hash,
			FileHash: hash,
			FilePath: destPath,
			Width:    w,
			Height:   h,
			SavedAt:  time.Now(),
		}); err == nil {
			imported++
			if imported%10 == 0 {
				glib.IdleAdd(func() {
					if activeMainWindow != nil {
						activeMainWindow.loadSavedStickersUI()
					}
				})
			}
		}
	}

	if imported > 0 {
		log.Printf("importStickersFromChatHistory: imported %d stickers from chat history", imported)
		glib.IdleAdd(func() {
			if activeMainWindow != nil {
				activeMainWindow.loadSavedStickersUI()
			}
		})
	}
	return imported
}

// MessageEdit represents a historical version of an edited message.
type MessageEdit struct {
	ID        int64
	MessageID string
	ChatJID   string
	Text      string
	EditedAt  time.Time
}

// saveMessageEdit archives the previous version of a message and updates it with the new text.
func (s *messageStore) saveMessageEdit(ctx context.Context, chatJID, messageID, newText string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var prevText string
	var prevTs int64
	if err := tx.QueryRowContext(ctx, `SELECT text, timestamp FROM whatsdown_messages WHERE chat_jid = ? AND id = ?`, chatJID, messageID).Scan(&prevText, &prevTs); err == nil && prevText != "" {
		_, _ = tx.ExecContext(ctx, `INSERT INTO whatsdown_message_edits (message_id, chat_jid, text, edited_at) VALUES (?, ?, ?, ?)`, messageID, chatJID, prevText, prevTs)
	}

	_, err = tx.ExecContext(ctx, `UPDATE whatsdown_messages SET text = ?, edited = 1 WHERE chat_jid = ? AND id = ?`, newText, chatJID, messageID)
	if err != nil {
		return err
	}
	return tx.Commit()
}

// getMessageEdits returns all archived edits of a message in chronological order.
func (s *messageStore) getMessageEdits(ctx context.Context, chatJID, messageID string) ([]MessageEdit, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, message_id, chat_jid, text, edited_at
		FROM whatsdown_message_edits
		WHERE chat_jid = ? AND message_id = ?
		ORDER BY edited_at ASC, id ASC
	`, chatJID, messageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var edits []MessageEdit
	for rows.Next() {
		var e MessageEdit
		var ts int64
		if err := rows.Scan(&e.ID, &e.MessageID, &e.ChatJID, &e.Text, &ts); err == nil {
			e.EditedAt = time.Unix(ts, 0)
			edits = append(edits, e)
		}
	}
	return edits, rows.Err()
}


