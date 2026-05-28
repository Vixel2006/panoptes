package repo

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Vixel2006/panoptes/internal/core/models"
)

type NoteRepository struct {
	db *sql.DB
}

func NewNoteRepository(db *sql.DB) *NoteRepository {
	return &NoteRepository{db: db}
}

func (r *NoteRepository) Create(ctx context.Context, n *model.Note) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO notes (id, title, body, group_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		n.ID, n.Title, n.Body, n.GroupID, timeToText(n.CreatedAt), timeToText(n.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("create note: %w", err)
	}
	return nil
}

func (r *NoteRepository) GetByID(ctx context.Context, id string) (*model.Note, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, title, body, group_id, created_at, updated_at FROM notes WHERE id = ?`, id,
	)
	n := &model.Note{}
	var createdAt, updatedAt string
	if err := row.Scan(&n.ID, &n.Title, &n.Body, &n.GroupID, &createdAt, &updatedAt); err != nil {
		return nil, fmt.Errorf("get note: %w", err)
	}
	n.CreatedAt, _ = textToTime(createdAt)
	n.UpdatedAt, _ = textToTime(updatedAt)
	return n, nil
}

func (r *NoteRepository) ListByGroup(ctx context.Context, groupID string) ([]*model.Note, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, title, body, group_id, created_at, updated_at FROM notes WHERE group_id = ? ORDER BY created_at ASC`,
		groupID,
	)
	if err != nil {
		return nil, fmt.Errorf("list notes: %w", err)
	}
	defer rows.Close()

	var out []*model.Note
	for rows.Next() {
		n := &model.Note{}
		var createdAt, updatedAt string
		if err := rows.Scan(&n.ID, &n.Title, &n.Body, &n.GroupID, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan note: %w", err)
		}
		n.CreatedAt, _ = textToTime(createdAt)
		n.UpdatedAt, _ = textToTime(updatedAt)
		out = append(out, n)
	}
	return out, rows.Err()
}

func (r *NoteRepository) Update(ctx context.Context, n *model.Note) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE notes SET title = ?, body = ?, updated_at = ? WHERE id = ?`,
		n.Title, n.Body, timeToText(n.UpdatedAt), n.ID,
	)
	if err != nil {
		return fmt.Errorf("update note: %w", err)
	}
	return nil
}

func (r *NoteRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM notes WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete note: %w", err)
	}
	return nil
}
