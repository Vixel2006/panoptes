package model

import (
	"encoding/json"
	"time"
)

type Response struct {
	ID         string          `json:"id"`
	Status     string          `json:"status"`
	StatusCode int             `json:"status_code"`
	Header     json.RawMessage `json:"header"`
	Payload    json.RawMessage `json:"payload"`
	Length     int64           `json:"length"`
	RequestID  string          `json:"request_id"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
