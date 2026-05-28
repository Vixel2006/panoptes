package repo

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/Vixel2006/panoptes/internal/core/models"
)

type ResponseRepository struct {
	db *sql.DB
}

func NewResponseRepository(db *sql.DB) *ResponseRepository {
	return &ResponseRepository{db: db}
}

func (r *ResponseRepository) Create(ctx context.Context, resp *model.Response) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO responses (id, status, status_code, header, payload, length, request_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		resp.ID, resp.Status, resp.StatusCode, string(resp.Header), []byte(resp.Payload), resp.Length, resp.RequestID, timeToText(resp.CreatedAt), timeToText(resp.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("create response: %w", err)
	}
	return nil
}

func (r *ResponseRepository) GetByID(ctx context.Context, id string) (*model.Response, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, status, status_code, header, payload, length, request_id, created_at, updated_at FROM responses WHERE id = ?`, id,
	)
	return scanResponse(row)
}

func (r *ResponseRepository) GetByRequestID(ctx context.Context, requestID string) (*model.Response, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, status, status_code, header, payload, length, request_id, created_at, updated_at FROM responses WHERE request_id = ?`, requestID,
	)
	return scanResponse(row)
}

func (r *ResponseRepository) ListByGroup(ctx context.Context, groupID string) ([]*model.Response, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT r.id, r.status, r.status_code, r.header, r.payload, r.length, r.request_id, r.created_at, r.updated_at
		FROM responses r
		JOIN requests req ON req.id = r.request_id
		WHERE req.group_id = ?
		ORDER BY r.created_at ASC`,
		groupID,
	)
	if err != nil {
		return nil, fmt.Errorf("list responses by group: %w", err)
	}
	defer rows.Close()

	var out []*model.Response
	for rows.Next() {
		resp, err := scanResponse(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, resp)
	}
	return out, rows.Err()
}

func (r *ResponseRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM responses WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete response: %w", err)
	}
	return nil
}

func scanResponse(row scannable) (*model.Response, error) {
	resp := &model.Response{}
	var headerStr, createdAt, updatedAt string
	var payload []byte
	if err := row.Scan(&resp.ID, &resp.Status, &resp.StatusCode, &headerStr, &payload, &resp.Length, &resp.RequestID, &createdAt, &updatedAt); err != nil {
		return nil, fmt.Errorf("scan response: %w", err)
	}
	resp.Header = json.RawMessage(headerStr)
	if payload != nil {
		resp.Payload = json.RawMessage(payload)
	}
	resp.CreatedAt, _ = textToTime(createdAt)
	resp.UpdatedAt, _ = textToTime(updatedAt)
	return resp, nil
}
