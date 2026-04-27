package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/lenchik-en/lbs_server/internal/domain/model"
)

type ExternalRepo struct {
	db *sql.DB
}

func NewExternalRepo(dsn string) (*ExternalRepo, error) {
	db, err := Open(dsn)
	if err != nil {
		return nil, fmt.Errorf("externalRepo: %w", err)
	}
	return &ExternalRepo{db: db}, nil
}

func (r *ExternalRepo) FindLTE(ctx context.Context, lte *model.LTE) (*model.Location, error) {
	const query = `
		SELECT lat, lon FROM cells
		WHERE radio = 3 AND mcc = $1 AND mnc = $2 AND area = $3 AND cell = $4
		LIMIT 1
	`
	return r.scan(r.db.QueryRowContext(ctx, query, lte.MCC, lte.MNC, lte.TAC, lte.CI))
}

func (r *ExternalRepo) FindGSM(ctx context.Context, gsm *model.GSM) (*model.Location, error) {
	const query = `
		SELECT lat, lon FROM cells
		WHERE radio = 1 AND mcc = $1 AND mnc = $2 AND area = $3 AND cell = $4
		LIMIT 1
	`
	return r.scan(r.db.QueryRowContext(ctx, query, gsm.MCC, gsm.MNC, gsm.LAC, gsm.CID))
}

func (r *ExternalRepo) FindWCDMA(ctx context.Context, wcdma *model.WCDMA) (*model.Location, error) {
	const query = `
		SELECT lat, lon FROM cells
		WHERE radio = 2 AND mcc = $1 AND mnc = $2 AND area = $3 AND cell = $4
		LIMIT 1
	`
	return r.scan(r.db.QueryRowContext(ctx, query, wcdma.MCC, wcdma.MNC, wcdma.LAC, wcdma.CID))
}

func (r *ExternalRepo) scan(row *sql.Row) (*model.Location, error) {
	var loc model.Location
	if err := row.Scan(&loc.Point.Lat, &loc.Point.Lon); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("external scan: %w", err)
	}
	return &loc, nil
}
