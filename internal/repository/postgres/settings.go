package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/lenchik-en/lbs_server/internal/domain/model"
)

type SettingsRepo struct {
	db *sql.DB
}

func NewSettingsRepo(db *sql.DB) *SettingsRepo {
	return &SettingsRepo{db: db}
}

func (r *SettingsRepo) Load(ctx context.Context) (*model.Settings, error) {
	var s model.Settings
	err := r.db.QueryRowContext(ctx, `
		SELECT require_key_locate, require_key_update, refresh_period_ms,
		       locate_min_period_key, locate_min_period_anon,
		       update_min_period_key, update_min_period_anon
		FROM server_settings WHERE id = 1
	`).Scan(
		&s.RequireKeyLocate, &s.RequireKeyUpdate, &s.RefreshPeriodMs,
		&s.LocateMinPeriodKey, &s.LocateMinPeriodAnon,
		&s.UpdateMinPeriodKey, &s.UpdateMinPeriodAnon,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load settings: %w", err)
	}
	return &s, nil
}

func (r *SettingsRepo) Save(ctx context.Context, s model.Settings) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO server_settings
		    (id, require_key_locate, require_key_update, refresh_period_ms,
		     locate_min_period_key, locate_min_period_anon,
		     update_min_period_key, update_min_period_anon)
		VALUES (1, $1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO UPDATE SET
		    require_key_locate   = EXCLUDED.require_key_locate,
		    require_key_update   = EXCLUDED.require_key_update,
		    refresh_period_ms    = EXCLUDED.refresh_period_ms,
		    locate_min_period_key  = EXCLUDED.locate_min_period_key,
		    locate_min_period_anon = EXCLUDED.locate_min_period_anon,
		    update_min_period_key  = EXCLUDED.update_min_period_key,
		    update_min_period_anon = EXCLUDED.update_min_period_anon
	`, s.RequireKeyLocate, s.RequireKeyUpdate, s.RefreshPeriodMs,
		s.LocateMinPeriodKey, s.LocateMinPeriodAnon,
		s.UpdateMinPeriodKey, s.UpdateMinPeriodAnon,
	)
	if err != nil {
		return fmt.Errorf("save settings: %w", err)
	}
	return nil
}
