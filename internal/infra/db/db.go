package db

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

type DB struct {
	*sql.DB
	Sessions  *SessionRepository
	Groups    *GroupRepository
	Requests  *RequestRepository
	Responses *ResponseRepository
	Notes     *NoteRepository
}

func Open(path string) (*DB, error) {
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	conn.SetMaxOpenConns(1)

	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}

	d := &DB{DB: conn}
	if err := d.migrate(); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}

	d.Sessions = &SessionRepository{db: d}
	d.Groups = &GroupRepository{db: d}
	d.Requests = &RequestRepository{db: d}
	d.Responses = &ResponseRepository{db: d}
	d.Notes = &NoteRepository{db: d}

	return d, nil
}

func (d *DB) Close() error {
	return d.DB.Close()
}
