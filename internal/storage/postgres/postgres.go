package postgres

import (
	"database/sql"
	"errors"
	"fmt"
	"url-shortener/internal/storage"

	"github.com/lib/pq"
)

type Storage struct {
	db *sql.DB
}

func New(connString string) (*Storage, error) {
	const op = "storage.postgres.New"

	db, err := sql.Open("postgres", connString)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	// Проверка соединения
	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS url(
    	id SERIAL PRIMARY KEY,
    	alias TEXT NOT NULL UNIQUE,
    	url TEXT NOT NULL UNIQUE 
	);

	CREATE INDEX IF NOT EXISTS idx_alias ON url(alias);
	CREATE INDEX IF NOT EXISTS idx_url ON url(url);
	`)

	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Storage{db: db}, nil
}

func (s *Storage) SaveURL(urlToSave string, alias string) (int64, error) {
	const op = "storage.postgres.SaveURL"

	stmt, err := s.db.Prepare(`INSERT INTO url (url, alias) VALUES ($1, $2) RETURNING id`)
	if err != nil {
		return 0, fmt.Errorf("%s: prepare statement: %w", op, err)
	}
	defer stmt.Close()

	var id int64
	err = stmt.QueryRow(urlToSave, alias).Scan(&id)
	if err != nil {
		// Проверка на нарушение уникальности
		var pgErr *pq.Error
		// if errors.As(err, &pgErr) && pgErr.Code == "23505" { // 23505 = unique_violation
		// 	// Проверяем, какое именно ограничение нарушено
		// 	if pgErr.Constraint == "url_unique" {
		// 		return 0, fmt.Errorf("%s: %w", op, storage.ErrURLExists)
		// 	}
		// 	// Если это ограничение на alias
		// 	if pgErr.Constraint == "url_alias_key" {
		// 		return 0, fmt.Errorf("%s: %w", op, storage.ErrURLExists)
		// 	}
		// }
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return 0, fmt.Errorf("%s: %w", op, storage.ErrURLExists)
		}
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	return id, nil
}

func (s *Storage) GetURL(alias string) (string, error) {
	const op = "storage.postgres.GetURL"

	stmt, err := s.db.Prepare("SELECT url FROM url WHERE alias = $1")
	if err != nil {
		return "", fmt.Errorf("%s: prepare statement: %w", op, err)
	}
	defer stmt.Close()

	var resURL string
	err = stmt.QueryRow(alias).Scan(&resURL)

	if errors.Is(err, sql.ErrNoRows) {
		return "", storage.ErrURLNotFound
	}

	if err != nil {
		return "", fmt.Errorf("%s: execute statement: %w", op, err)
	}

	return resURL, nil
}

func (s *Storage) Close() error {
	return s.db.Close()
}
