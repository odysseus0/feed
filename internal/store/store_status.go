package store

import (
	"context"
	"database/sql"
)

func (s *Store) ensureEntryStatus(ctx context.Context, id int64) error {
	if err := s.ensureEntryExists(ctx, id); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, `INSERT OR IGNORE INTO entry_status(entry_id) VALUES (?)`, id)
	return err
}

func (s *Store) ensureEntryExists(ctx context.Context, id int64) error {
	var exists int
	if err := s.db.QueryRowContext(ctx, `SELECT 1 FROM entries WHERE id = ?`, id).Scan(&exists); err != nil {
		if err == sql.ErrNoRows {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (s *Store) UpdateEntryRead(ctx context.Context, id int64, read bool) error {
	if err := s.ensureEntryStatus(ctx, id); err != nil {
		return err
	}
	query := `UPDATE entry_status SET read = 0, read_at = NULL WHERE entry_id = ?`
	if read {
		query = `UPDATE entry_status SET read = 1, read_at = CURRENT_TIMESTAMP WHERE entry_id = ?`
	}
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

func (s *Store) ToggleEntryStarred(ctx context.Context, id int64) (bool, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err = tx.ExecContext(ctx, `INSERT OR IGNORE INTO entry_status(entry_id) VALUES (?)`, id); err != nil {
		return false, err
	}

	var current bool
	if err = tx.QueryRowContext(ctx, `SELECT COALESCE(starred, 0) FROM entry_status WHERE entry_id = ?`, id).Scan(&current); err != nil {
		return false, err
	}
	next := !current

	query := `UPDATE entry_status SET starred = 0, starred_at = NULL WHERE entry_id = ?`
	if next {
		query = `UPDATE entry_status SET starred = 1, starred_at = CURRENT_TIMESTAMP WHERE entry_id = ?`
	}
	if _, err = tx.ExecContext(ctx, query, id); err != nil {
		return false, err
	}
	if err = tx.Commit(); err != nil {
		return false, err
	}
	return next, nil
}

func (s *Store) SetEntriesRead(ctx context.Context, ids []int64, read bool) error {
	query := `UPDATE entry_status SET read = 0, read_at = NULL WHERE entry_id = ?`
	if read {
		query = `UPDATE entry_status SET read = 1, read_at = CURRENT_TIMESTAMP WHERE entry_id = ?`
	}
	return s.batchUpdateEntryStatus(ctx, ids, query)
}

func (s *Store) SetEntriesStarred(ctx context.Context, ids []int64, starred bool) error {
	query := `UPDATE entry_status SET starred = 0, starred_at = NULL WHERE entry_id = ?`
	if starred {
		query = `UPDATE entry_status SET starred = 1, starred_at = CURRENT_TIMESTAMP WHERE entry_id = ?`
	}
	return s.batchUpdateEntryStatus(ctx, ids, query)
}

func (s *Store) batchUpdateEntryStatus(ctx context.Context, ids []int64, updateQuery string) (err error) {
	if len(ids) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	insStmt, err := tx.PrepareContext(ctx, `INSERT OR IGNORE INTO entry_status(entry_id) VALUES (?)`)
	if err != nil {
		return err
	}
	defer insStmt.Close()

	updStmt, err := tx.PrepareContext(ctx, updateQuery)
	if err != nil {
		return err
	}
	defer updStmt.Close()

	for _, id := range ids {
		var exists int
		if err = tx.QueryRowContext(ctx, `SELECT 1 FROM entries WHERE id = ?`, id).Scan(&exists); err != nil {
			if err == sql.ErrNoRows {
				return ErrNotFound
			}
			return err
		}
		if _, err = insStmt.ExecContext(ctx, id); err != nil {
			return err
		}
		if _, err = updStmt.ExecContext(ctx, id); err != nil {
			return err
		}
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}
