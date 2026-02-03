package sqlite

import (
	"database/sql"
	"errors"
	"fmt"
	"url-shortener/internal/storage"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

type Storage struct {
	db *sql.DB
}

func New(storagePath string) (*Storage, error) {
	const op = "storage.sqlite.New"

	db, err := sql.Open("sqlite", storagePath)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	stmt, err := db.Prepare(`
	CREATE TABLE IF NOT EXISTS urls (
    	id Int PRIMARY KEY,
    	alias TEXT NOT NULL UNIQUE,
    	url TEXT NOT NULL);
	CREATE INDEX IF NOT EXISTS idx_alias ON urls(alias);
    `)

	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()

	_, err = stmt.Exec()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Storage{db: db}, nil
}

func (s *Storage) SaveURL(urlToSave string, alias string) (int64, error) {
	const op = "storage.sqlite.SaveURL"
	stmt, err := s.db.Prepare("INSERT INTO urls (url, alias) VALUES (?, ?)")
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()

	res, err := stmt.Exec(urlToSave, alias)
	if err != nil {

		if sqlErr, ok := err.(*sqlite.Error); ok && sqlErr.Code() == sqlite3.SQLITE_CONSTRAINT {
			return 0, fmt.Errorf("%s: %w", op, storage.ErrURLExists)
		}

		return 0, fmt.Errorf("%s: %w", op, err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("%s: failed to get las inserted id: %w", op, err)
	}

	return id, nil
}

func (s *Storage) GetURL(alias string) (string, error) {
	const op = "storage.sqlite.GetURL"

	stmt, err := s.db.Prepare("SELECT url FROM urls WHERE alias = ?")
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()

	var res string
	err = stmt.QueryRow(alias).Scan(&res)

	if errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("%s: %w", op, storage.ErrNotFound)
	}

	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}

	return res, nil
}

func (s *Storage) DeleteURL(alias string) error {
	const op = "storage.sqlite.DeleteURL"

	stmt, err := s.db.Prepare("DELETE FROM urls WHERE alias = ?")

	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()

	res, err := stmt.Exec(alias)

	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("%s: %w", op, storage.ErrNotFound)
	}

	return nil
}
