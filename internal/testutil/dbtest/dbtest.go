package dbtest

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func Open(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	// PRAGMA must be set per-connection for modernc.org/sqlite
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatal(err)
	}

	migrations := []string{
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

	for _, m := range migrations {
		if _, err := db.Exec(m); err != nil {
			t.Fatal(err)
		}
	}

	return db
}
