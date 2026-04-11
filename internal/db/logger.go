package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

type SessionManager interface {
	CreateSessionIfNotExists(ctx context.Context, sessionUUID string) error
	SavePoint(ctx context.Context, sessionUUID string, lat, lon float64, accuracy int, source string, rawReq, rawResp interface{}) error
}

type Session struct {
	db *sql.DB
}

func NewLogger(db *sql.DB) *Session {
	return &Session{db: db}
}

func (s *Session) CreateSessionIfNotExists(ctx context.Context, sessionUUID string) error {
	_, err := s.db.ExecContext(ctx,
		`		
		INSERT INTO sessions (session_uuid)
		VALUES ($1)
		ON CONFLICT (session_uuid) DO NOTHING;
	`, sessionUUID)
	return err
}

func (s *Session) SavePoint(ctx context.Context, sessionUUID string, lat float64, lon float64, accuracy int, source string, rawReq interface{}, rawResp interface{}) error {
	var rawReqJSON, rawRespJSON []byte
	var err error

	if rawReq != nil {
		rawReqJSON, err = json.Marshal(rawReq)
		if err != nil {
			return fmt.Errorf("failed to marshall rawReq: %w", err)
		}
	}

	if rawResp != nil {
		rawRespJSON, err = json.Marshal(rawResp)
		if err != nil {
			return fmt.Errorf("failed to marshal rawResp: %w", err)
		}
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO session_points (
			session_uuid,
			timestamp,
			lat,
			lon,
			accuracy,
			source,
			raw_request,
			raw_response
		)
		VALUES ($1, NOW(), $2, $3, $4, $5, $6, $7)
        ON CONFLICT DO NOTHING;
	`,
		sessionUUID,
		lat,
		lon,
		accuracy,
		source,
		rawReqJSON,
		rawRespJSON,
	)

	return err
}
