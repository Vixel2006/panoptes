package model

import (
	"encoding/json"
	"time"
)

type Request struct {
	ID      string          `json:"id"`
	URL     string          `json:"url"`
	Method  string          `json:"method"`
	Header  json.RawMessage `json:"header"`
	Payload json.RawMessage `json:"payload"`
	Length  int64           `json:"length"`
	GroupID   string          `json:"group_id,omitempty"`
	SessionID string          `json:"session_id,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
