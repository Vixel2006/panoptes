package db

import "time"

const timeFormat = time.RFC3339Nano

func timeToText(t time.Time) string {
	return t.UTC().Format(timeFormat)
}

func textToTime(s string) (time.Time, error) {
	return time.Parse(timeFormat, s)
}
