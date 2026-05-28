package db

import (
	"context"
	"fmt"

	"github.com/Vixel2006/panoptes/internal/core/models"
)

type SessionRepository struct {
	db *DB
}

func (r *SessionRepository) Create(ctx context.Context, s *model.Session) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO sessions (id, name, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		s.ID, s.Name, timeToText(s.CreatedAt), timeToText(s.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	return nil
}

func (r *SessionRepository) GetByID(ctx context.Context, id string) (*model.Session, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, name, created_at, updated_at FROM sessions WHERE id = ?`, id,
	)
	s := &model.Session{}
	var createdAt, updatedAt string
	if err := row.Scan(&s.ID, &s.Name, &createdAt, &updatedAt); err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}
	s.CreatedAt, _ = textToTime(createdAt)
	s.UpdatedAt, _ = textToTime(updatedAt)
	return s, nil
}

func (r *SessionRepository) List(ctx context.Context) ([]*model.Session, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, created_at, updated_at FROM sessions ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	defer rows.Close()

	var out []*model.Session
	for rows.Next() {
		s := &model.Session{}
		var createdAt, updatedAt string
		if err := rows.Scan(&s.ID, &s.Name, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}
		s.CreatedAt, _ = textToTime(createdAt)
		s.UpdatedAt, _ = textToTime(updatedAt)
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *SessionRepository) Update(ctx context.Context, s *model.Session) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE sessions SET name = ?, updated_at = ? WHERE id = ?`,
		s.Name, timeToText(s.UpdatedAt), s.ID,
	)
	if err != nil {
		return fmt.Errorf("update session: %w", err)
	}
	return nil
}

func (r *SessionRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}
