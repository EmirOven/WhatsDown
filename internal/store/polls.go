package store

import "context"

func (s *MessageStore) GetPollResults(ctx context.Context, chatJID, messageID string) (*PollResults, error) {
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

func (s *MessageStore) GetPollVotes(ctx context.Context, chatJID, messageID string, voterJID ...string) (map[string]bool, error) {
	voter := "me"
	if len(voterJID) > 0 && voterJID[0] != "" {
		voter = voterJID[0]
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT option_name FROM whatsdown_poll_votes
		WHERE chat_jid = ? AND message_id = ? AND voter_jid = ?`,
		chatJID, messageID, voter)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	votes := make(map[string]bool)
	for rows.Next() {
		var opt string
		if err := rows.Scan(&opt); err == nil {
			votes[opt] = true
		}
	}
	return votes, nil
}

func (s *MessageStore) SetPollVotesForVoter(ctx context.Context, chatJID, messageID, voterJID string, options []string) error {
	voter := "me"
	if voterJID != "" {
		voter = voterJID
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		DELETE FROM whatsdown_poll_votes
		WHERE chat_jid = ? AND message_id = ? AND voter_jid = ?`,
		chatJID, messageID, voter)
	if err != nil {
		return err
	}

	for _, opt := range options {
		if opt == "" {
			continue
		}
		_, err = tx.ExecContext(ctx, `
			INSERT OR IGNORE INTO whatsdown_poll_votes (chat_jid, message_id, voter_jid, option_name)
			VALUES (?, ?, ?, ?)`,
			chatJID, messageID, voter, opt)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}
