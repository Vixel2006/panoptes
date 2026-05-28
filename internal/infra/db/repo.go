package db

import (
	"database/sql"
	"time"
)

const timeFormat = time.RFC3339Nano

func timeToText(t time.Time) string {
	return t.UTC().Format(timeFormat)
}

func textToTime(s string) (time.Time, error) {
	return time.Parse(timeFormat, s)
}

func nullStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func strPtr(s sql.NullString) string {
	if s.Valid {
		return s.String
	}
	return ""
}
