package postgres

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/lenchik-en/lbs_server/internal/domain/model"
)

type APIKeyRepo struct {
	db *sql.DB
}

func NewAPIKeyRepo(db *sql.DB) *APIKeyRepo {
	return &APIKeyRepo{db: db}
}

func (r *APIKeyRepo) GetByKey(ctx context.Context, key string) (*model.APIKey, error) {
	const query = `
		SELECT key, scope, is_active, created_at
		FROM api_keys
		WHERE key = $1
	`
	row := r.db.QueryRowContext(ctx, query, key)
	var k model.APIKey
	if err := row.Scan(&k.Key, &k.Scope, &k.IsActive, &k.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("GetByKey scan: %w", err)
	}
	return &k, nil
}

func (r *APIKeyRepo) Create(ctx context.Context, scope model.KeyScope) (*model.APIKey, error) {
	key, err := generateKey(scope)
	if err != nil {
		return nil, err
	}
	const query = `
		INSERT INTO api_keys (key, scope, is_active, created_at)
		VALUES ($1, $2, true, $3)
	`
	now := time.Now()
	if _, err := r.db.ExecContext(ctx, query, key, scope, now); err != nil {
		return nil, fmt.Errorf("Create api key: %w", err)
	}
	return &model.APIKey{Key: key, Scope: scope, IsActive: true, CreatedAt: now}, nil
}

func (r *APIKeyRepo) Deactivate(ctx context.Context, key string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE api_keys SET is_active = false WHERE key = $1`, key)
	return err
}

func (r *APIKeyRepo) ListAll(ctx context.Context) ([]*model.APIKey, error) {
	const query = `SELECT key, scope, is_active, created_at FROM api_keys ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*model.APIKey
	for rows.Next() {
		var k model.APIKey
		if err := rows.Scan(&k.Key, &k.Scope, &k.IsActive, &k.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, &k)
	}
	return result, rows.Err()
}

// generateKey формирует ключ вида lbs_locate_<32 hex> или lbs_update_<32 hex>.
func generateKey(scope model.KeyScope) (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("rand: %w", err)
	}
	return fmt.Sprintf("lbs_%s_%s", scope, hex.EncodeToString(b)), nil
}
