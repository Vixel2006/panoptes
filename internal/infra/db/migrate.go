package db

import "fmt"

var migrations = []string{
	`CREATE TABLE IF NOT EXISTS sessions (
		id        TEXT PRIMARY KEY,
		name      TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	)`,
	`CREATE TABLE IF NOT EXISTS groups (
		id         TEXT PRIMARY KEY,
		session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	)`,
	`CREATE TABLE IF NOT EXISTS requests (
		id         TEXT PRIMARY KEY,
		url        TEXT NOT NULL,
		method     TEXT NOT NULL,
		header     TEXT NOT NULL DEFAULT '{}',
		payload    BLOB,
		length     INTEGER NOT NULL DEFAULT 0,
		group_id   TEXT REFERENCES groups(id) ON DELETE SET NULL,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	)`,
	`CREATE TABLE IF NOT EXISTS responses (
		id          TEXT PRIMARY KEY,
		status      TEXT NOT NULL,
		status_code INTEGER NOT NULL,
		header      TEXT NOT NULL DEFAULT '{}',
		payload     BLOB,
		length      INTEGER NOT NULL DEFAULT 0,
		request_id  TEXT NOT NULL REFERENCES requests(id) ON DELETE CASCADE,
		created_at  TEXT NOT NULL,
		updated_at  TEXT NOT NULL
	)`,
	`CREATE TABLE IF NOT EXISTS notes (
		id         TEXT PRIMARY KEY,
		title      TEXT NOT NULL DEFAULT '',
		body       TEXT NOT NULL DEFAULT '',
		group_id   TEXT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	)`,
}

func (d *DB) migrate() error {
	for i, m := range migrations {
		if _, err := d.Exec(m); err != nil {
			return fmt.Errorf("migration %d: %w", i, err)
		}
	}
	return nil
}
