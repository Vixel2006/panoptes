package model

import "time"

type Group struct {
	ID        string    `json:"id"`
	SessionID string    `json:"session_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
