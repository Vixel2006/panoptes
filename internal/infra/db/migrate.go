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
		name       TEXT NOT NULL DEFAULT '',
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
		session_id TEXT REFERENCES sessions(id) ON DELETE CASCADE,
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

	// Ensure name column exists in groups table
	var groupNameCount int
	err := d.QueryRow("SELECT count(*) FROM pragma_table_info('groups') WHERE name='name'").Scan(&groupNameCount)
	if err == nil && groupNameCount == 0 {
		_, _ = d.Exec("ALTER TABLE groups ADD COLUMN name TEXT NOT NULL DEFAULT ''")
	}

	// Ensure session_id column exists in requests table
	var count int
	err = d.QueryRow("SELECT count(*) FROM pragma_table_info('requests') WHERE name='session_id'").Scan(&count)
	if err == nil && count == 0 {
		_, _ = d.Exec("ALTER TABLE requests ADD COLUMN session_id TEXT REFERENCES sessions(id) ON DELETE CASCADE")
	}

	return nil
}
