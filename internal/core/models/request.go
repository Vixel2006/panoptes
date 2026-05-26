package model

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"time"
)

type Request struct {
	ID      string          `json:"id"`
	URL     string          `json:"url"`
	Method  string          `json:"method"`
	Header  json.RawMessage `json:"header"`
	Payload json.RawMessage `json:"payload"`
	Length  int64           `json:"length"`
	GroupID string          `json:"group_id,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Runtime fields — not serialised to JSON / not stored in DB.
	ParsedURL *url.URL      `json:"-"`
	Host      string        `json:"-"`
	Headers   http.Header   `json:"-"`
	Body      io.ReadCloser `json:"-"`
	RawBody   []byte        `json:"-"`
}
