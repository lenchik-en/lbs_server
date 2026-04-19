package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

type SessionRepo struct {
	db *sql.DB
}

func NewSessionRepo(db *sql.DB) *SessionRepo {
	return &SessionRepo{db: db}
}

func (r *SessionRepo) CreateIfNotExists(ctx context.Context, sessionUUID string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO sessions (session_uuid)
		VALUES ($1)
		ON CONFLICT (session_uuid) DO NOTHING
	`, sessionUUID)
	return err
}

func (r *SessionRepo) SavePoint(ctx context.Context, sessionUUID string, lat, lon float64, accuracy int, source string, rawReq, rawResp any) error {
	reqJSON, err := json.Marshal(rawReq)
	if err != nil {
		return fmt.Errorf("marshal rawReq: %w", err)
	}
	respJSON, err := json.Marshal(rawResp)
	if err != nil {
		return fmt.Errorf("marshal rawResp: %w", err)
	}
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO session_points (session_uuid, timestamp, lat, lon, accuracy, source, raw_request, raw_response)
		VALUES ($1, NOW(), $2, $3, $4, $5, $6, $7)
	`, sessionUUID, lat, lon, accuracy, source, reqJSON, respJSON)
	return err
}
