package store

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

func Open(path string) (*sql.DB, error) {
	dsn := fmt.Sprintf("%s?_journal_mode=WAL&_busy_timeout=5000", path)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}
	if err := migrate(db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func migrate(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		CREATE TABLE IF NOT EXISTS events (
			id              TEXT PRIMARY KEY,
			hook            TEXT NOT NULL,
			occurred_at     INTEGER NOT NULL,
			skipped         INTEGER NOT NULL DEFAULT 0,
			commit_msg      TEXT,
			related_commit  TEXT,
			commit_hash     TEXT
		);

		CREATE TABLE IF NOT EXISTS answers (
			id         TEXT PRIMARY KEY,
			event_id   TEXT NOT NULL REFERENCES events(id),
			question   TEXT NOT NULL,
			answer     TEXT NOT NULL,
			tag        TEXT NOT NULL DEFAULT ''
		);

		CREATE TABLE IF NOT EXISTS complexity_history (
			id          TEXT PRIMARY KEY,
			path        TEXT NOT NULL,
			score       REAL NOT NULL,
			recorded_at INTEGER NOT NULL
		);
	`)
	if err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}

	// Add new columns to existing databases; errors mean column already exists — safe to ignore.
	db.Exec(`ALTER TABLE answers ADD COLUMN tag TEXT NOT NULL DEFAULT ''`)
	db.Exec(`ALTER TABLE events ADD COLUMN related_commit TEXT`)
	db.Exec(`ALTER TABLE events ADD COLUMN commit_hash TEXT`)
	return nil
}
