package store

import "context"

func (s *MessageStore) TogglePollVote(ctx context.Context, chatJID, messageID, optionName string, allowMultiple bool, voterJID ...string) (map[string]bool, error) {
	voter := "me"
	if len(voterJID) > 0 && voterJID[0] != "" {
		voter = voterJID[0]
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var exists int
	err = tx.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM whatsdown_poll_votes
		WHERE chat_jid = ? AND message_id = ? AND voter_jid = ? AND option_name = ?`,
		chatJID, messageID, voter, optionName).Scan(&exists)
	if err != nil {
		return nil, err
	}

	if exists > 0 {
		_, err = tx.ExecContext(ctx, `
			DELETE FROM whatsdown_poll_votes
			WHERE chat_jid = ? AND message_id = ? AND voter_jid = ? AND option_name = ?`,
			chatJID, messageID, voter, optionName)
		if err != nil {
			return nil, err
		}
	} else {
		if !allowMultiple {
			_, err = tx.ExecContext(ctx, `
				DELETE FROM whatsdown_poll_votes
				WHERE chat_jid = ? AND message_id = ? AND voter_jid = ?`,
				chatJID, messageID, voter)
			if err != nil {
				return nil, err
			}
		}
		_, err = tx.ExecContext(ctx, `
			INSERT OR IGNORE INTO whatsdown_poll_votes (chat_jid, message_id, voter_jid, option_name)
			VALUES (?, ?, ?, ?)`,
			chatJID, messageID, voter, optionName)
		if err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return s.GetPollVotes(ctx, chatJID, messageID, voter)
}
