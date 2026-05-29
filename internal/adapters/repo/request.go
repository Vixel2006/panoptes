package repo

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/Vixel2006/panoptes/internal/core/models"
)

type RequestRepository struct {
	db *sql.DB
}

func NewRequestRepository(db *sql.DB) *RequestRepository {
	return &RequestRepository{db: db}
}

func (r *RequestRepository) Create(ctx context.Context, req *model.Request) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO requests (id, url, method, header, payload, length, group_id, session_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		req.ID, req.URL, req.Method, string(req.Header), []byte(req.Payload), req.Length, nullStr(req.GroupID), nullStr(req.SessionID), timeToText(req.CreatedAt), timeToText(req.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	return nil
}

func (r *RequestRepository) GetByID(ctx context.Context, id string) (*model.Request, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, url, method, header, payload, length, group_id, session_id, created_at, updated_at FROM requests WHERE id = ?`, id,
	)
	return scanRequest(row)
}

func (r *RequestRepository) ListByGroup(ctx context.Context, groupID string) ([]*model.Request, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, url, method, header, payload, length, group_id, session_id, created_at, updated_at FROM requests WHERE group_id = ? ORDER BY created_at ASC`,
		groupID,
	)
	if err != nil {
		return nil, fmt.Errorf("list requests by group: %w", err)
	}
	defer rows.Close()

	var out []*model.Request
	for rows.Next() {
		req, err := scanRequest(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, req)
	}
	return out, rows.Err()
}

func (r *RequestRepository) ListBySession(ctx context.Context, sessionID string) ([]*model.Request, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, url, method, header, payload, length, group_id, session_id, created_at, updated_at FROM requests WHERE session_id = ? ORDER BY created_at ASC`,
		sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("list requests by session: %w", err)
	}
	defer rows.Close()

	var out []*model.Request
	for rows.Next() {
		req, err := scanRequest(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, req)
	}
	return out, rows.Err()
}

func (r *RequestRepository) ListAll(ctx context.Context) ([]*model.Request, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, url, method, header, payload, length, group_id, session_id, created_at, updated_at FROM requests ORDER BY created_at ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list all requests: %w", err)
	}
	defer rows.Close()

	var out []*model.Request
	for rows.Next() {
		req, err := scanRequest(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, req)
	}
	return out, rows.Err()
}

func (r *RequestRepository) UpdateGroup(ctx context.Context, id, groupID string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE requests SET group_id = ? WHERE id = ?`, nullStr(groupID), id,
	)
	if err != nil {
		return fmt.Errorf("update request group: %w", err)
	}
	return nil
}

func (r *RequestRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM requests WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete request: %w", err)
	}
	return nil
}

type scannable interface {
	Scan(dest ...any) error
}

func scanRequest(row scannable) (*model.Request, error) {
	req := &model.Request{}
	var headerStr, createdAt, updatedAt string
	var payload []byte
	var groupID, sessionID *string
	if err := row.Scan(&req.ID, &req.URL, &req.Method, &headerStr, &payload, &req.Length, &groupID, &sessionID, &createdAt, &updatedAt); err != nil {
		return nil, fmt.Errorf("scan request: %w", err)
	}
	req.Header = json.RawMessage(headerStr)
	if payload != nil {
		req.Payload = json.RawMessage(payload)
	}
	if groupID != nil {
		req.GroupID = *groupID
	}
	if sessionID != nil {
		req.SessionID = *sessionID
	}
	req.CreatedAt, _ = textToTime(createdAt)
	req.UpdatedAt, _ = textToTime(updatedAt)
	return req, nil
}
