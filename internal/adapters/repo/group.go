package repo

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Vixel2006/panoptes/internal/core/models"
)

type GroupRepository struct {
	db *sql.DB
}

func NewGroupRepository(db *sql.DB) *GroupRepository {
	return &GroupRepository{db: db}
}

func (r *GroupRepository) Create(ctx context.Context, g *model.Group) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO groups (id, session_id, name, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		g.ID, g.SessionID, g.Name, timeToText(g.CreatedAt), timeToText(g.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("create group: %w", err)
	}
	return nil
}

func (r *GroupRepository) GetByID(ctx context.Context, id string) (*model.Group, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, session_id, name, created_at, updated_at FROM groups WHERE id = ?`, id,
	)
	g := &model.Group{}
	var createdAt, updatedAt string
	if err := row.Scan(&g.ID, &g.SessionID, &g.Name, &createdAt, &updatedAt); err != nil {
		return nil, fmt.Errorf("get group: %w", err)
	}
	g.CreatedAt, _ = textToTime(createdAt)
	g.UpdatedAt, _ = textToTime(updatedAt)
	return g, nil
}

func (r *GroupRepository) ListBySession(ctx context.Context, sessionID string) ([]*model.Group, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, session_id, name, created_at, updated_at FROM groups WHERE session_id = ? ORDER BY created_at ASC`,
		sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("list groups: %w", err)
	}
	defer rows.Close()

	var out []*model.Group
	for rows.Next() {
		g := &model.Group{}
		var createdAt, updatedAt string
		if err := rows.Scan(&g.ID, &g.SessionID, &g.Name, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan group: %w", err)
		}
		g.CreatedAt, _ = textToTime(createdAt)
		g.UpdatedAt, _ = textToTime(updatedAt)
		out = append(out, g)
	}
	return out, rows.Err()
}

func (r *GroupRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM groups WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete group: %w", err)
	}
	return nil
}
