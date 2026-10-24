package store

import (
	"context"
	"database/sql"
	"fmt"
)

var migrations = []string{
	migration1, migration2, migration3, migration4, migration5,
	migration6, migration7, migration8, migration9, migration10,
	migration11, migration12, migration13, migration14, migration15, migration16, migration17, migration18,
}

func (s *MessageStore) migrate(ctx context.Context) error {
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
