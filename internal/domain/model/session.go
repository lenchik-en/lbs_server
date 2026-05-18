package model

import "time"

type SessionListRow struct {
	SessionUUID string
	StartedAt   time.Time
	PointCount  int
	LastSeen    *time.Time
}

type SessionPoint struct {
	ID         int64
	Timestamp  time.Time
	Lat        float64
	Lon        float64
	Accuracy   int
	Source     string
	RawRequest []byte // JSONB из session_points.raw_request
}
