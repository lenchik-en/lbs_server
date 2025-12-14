package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

type Logger struct {
	db *sql.DB
}

func NewLogger(db *sql.DB) *Logger {
	return &Logger{db: db}
}

// CreateSessionIfNotExists - создает сессию, если ее нет
func (l *Logger) CreateSessionIfNotExists(ctx context.Context, sessionUUID string) error {
	_, err := l.db.ExecContext(ctx,
		`		
		INSERT INTO sessions (session_uuid)
		VALUES ($1)
		ON CONFLICT (session_uuid) DO NOTHING;
	`, sessionUUID)
	return err
}

func (l *Logger) SavePoint(ctx context.Context, sessionUUID string, lat float64, lon float64, accuracy int, source string, rawReq interface{}, rawResp interface{}) error {
	var rawReqJSON, rawRespJSON []byte
	var err error

	if rawReq != nil {
		rawReqJSON, err = json.Marshal(rawReq)
		if err != nil {
			return fmt.Errorf("failed to marshall rawReq: %w", err)
		}
	}

	if rawResp != nil {
		rawReqJSON, err = json.Marshal(rawResp)
		if err != nil {
			return fmt.Errorf("failed to marshal rawResp: %w", err)
		}
	}

	_, err = l.db.ExecContext(ctx, `
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
		VALUES ($1, NOW(), $2, $3, $4, $5, $6, $7);
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
