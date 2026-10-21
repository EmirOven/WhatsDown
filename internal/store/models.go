package store

import "time"

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

// Chat represents a conversation thread in the sidebar.
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
	LastMessageKind   string
	LastMessageFromMe bool
	LastMessageStatus int
}

// Reaction represents an emoji reaction on a message.
type Reaction struct {
	ChatJID    string
	MessageID  string
	SenderJID  string
	SenderName string
	Emoji      string
	FromMe     bool
	Timestamp  time.Time
}

// StoredMessage is a single message retrieved from SQLite.
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
	Starred     bool
	Status      int
	SenderSaved bool
	Reactions   []Reaction
	Raw         []byte
}

// PollResults stores aggregate vote counts for a poll.
type PollResults struct {
	TotalVoters    int
	VoteCounts     map[string]int
	VotersByOption map[string][]string
}

// SavedSticker represents a sticker saved in the local sticker gallery.
type SavedSticker struct {
	ID       string
	FilePath string
	FileHash string
	Width    int
	Height   int
	SavedAt  time.Time
}

// MessageEdit represents an entry in a message's edit history.
type MessageEdit struct {
	ID        int64
	MessageID string
	ChatJID   string
	Text      string
	EditedAt  time.Time
}
