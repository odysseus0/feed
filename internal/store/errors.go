package store

import (
	"database/sql"
	"errors"
	"fmt"
)

var (
	ErrNotFound     = errors.New("not found")
	ErrInvalidInput = errors.New("invalid input")
)

func wrapNotFound(entity string, err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("%s: %w", entity, ErrNotFound)
	}
	return err
}
