package repository

import (
	"database/sql"
	"encoding/json"
	"time"
)

func nowString() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

func parseTime(raw string) time.Time {
	t, _ := time.Parse(time.RFC3339Nano, raw)
	return t
}

func nullableString(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	return &ns.String
}

func nullableTime(ns sql.NullString) *time.Time {
	if !ns.Valid {
		return nil
	}
	t := parseTime(ns.String)
	return &t
}

func nullableInt64(ni sql.NullInt64) *int64 {
	if !ni.Valid {
		return nil
	}
	return &ni.Int64
}

func originsJSON(origins []string) string {
	b, _ := json.Marshal(origins)
	return string(b)
}

func parseOrigins(raw string) []string {
	var origins []string
	if err := json.Unmarshal([]byte(raw), &origins); err != nil {
		return nil
	}
	return origins
}
