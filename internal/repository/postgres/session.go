package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/lenchik-en/lbs_server/internal/domain/model"
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

func (r *SessionRepo) SavePoint(ctx context.Context, sessionUUID string, lat, lon float64, accuracy float64, source string, rawReq, rawResp any) error {
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

func (r *SessionRepo) ListSessions(ctx context.Context, limit, offset int) ([]model.SessionListRow, int64, error) {
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sessions`).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count sessions: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT s.session_uuid, s.started_at,
		       COUNT(sp.id)          AS point_count,
		       MAX(sp.timestamp)     AS last_seen
		FROM sessions s
		LEFT JOIN session_points sp ON sp.session_uuid = s.session_uuid
		GROUP BY s.session_uuid, s.started_at
		ORDER BY s.started_at DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list sessions: %w", err)
	}
	defer rows.Close()

	var result []model.SessionListRow
	for rows.Next() {
		var row model.SessionListRow
		if err := rows.Scan(&row.SessionUUID, &row.StartedAt, &row.PointCount, &row.LastSeen); err != nil {
			return nil, 0, fmt.Errorf("scan session row: %w", err)
		}
		result = append(result, row)
	}
	return result, total, rows.Err()
}

func (r *SessionRepo) GetSessionPoints(ctx context.Context, sessionUUID string) ([]model.SessionPoint, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, timestamp, lat, lon, accuracy, source, raw_request
		FROM session_points
		WHERE session_uuid = $1
		ORDER BY timestamp ASC
	`, sessionUUID)
	if err != nil {
		return nil, fmt.Errorf("get session points: %w", err)
	}
	defer rows.Close()

	var result []model.SessionPoint
	for rows.Next() {
		var p model.SessionPoint
		if err := rows.Scan(&p.ID, &p.Timestamp, &p.Lat, &p.Lon, &p.Accuracy, &p.Source, &p.RawRequest); err != nil {
			return nil, fmt.Errorf("scan session point: %w", err)
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

func (r *SessionRepo) UpdatePoint(ctx context.Context, id int64, lat, lon float64, accuracy float64) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE session_points SET lat = $1, lon = $2, accuracy = $3 WHERE id = $4
	`, lat, lon, accuracy, id)
	if err != nil {
		return fmt.Errorf("update point: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("point %d not found", id)
	}
	return nil
}
