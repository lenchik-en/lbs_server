package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/lenchik-en/lbs_server/internal/domain/model"
)

type LocateRepo struct {
	db *sql.DB
}

func NewLocateRepo(dsn string) (*LocateRepo, error) {
	db, err := Open(dsn)
	if err != nil {
		return nil, fmt.Errorf("locateRepo: %w", err)
	}
	return &LocateRepo{db: db}, nil
}

func (r *LocateRepo) DB() *sql.DB { return r.db }

func (r *LocateRepo) FindLTE(ctx context.Context, lte *model.LTE) (*model.Location, error) {
	const query = `
		SELECT lat, lon, from_metro
		FROM cells
		WHERE tech = 'LTE' AND mcc = $1 AND mnc = $2 AND tac = $3 AND ci = $4
		LIMIT 1
	`
	return r.scanLocation(r.db.QueryRowContext(ctx, query, lte.MCC, lte.MNC, lte.TAC, lte.CI))
}

func (r *LocateRepo) FindGSM(ctx context.Context, gsm *model.GSM) (*model.Location, error) {
	const query = `
		SELECT lat, lon, from_metro
		FROM cells
		WHERE tech = 'GSM' AND mcc = $1 AND mnc = $2 AND lac = $3 AND cid = $4
		LIMIT 1
	`
	return r.scanLocation(r.db.QueryRowContext(ctx, query, gsm.MCC, gsm.MNC, gsm.LAC, gsm.CID))
}

func (r *LocateRepo) FindWCDMA(ctx context.Context, wcdma *model.WCDMA) (*model.Location, error) {
	const query = `
		SELECT lat, lon, from_metro
		FROM cells
		WHERE tech = 'WCDMA' AND mcc = $1 AND mnc = $2 AND psc = $3 AND cid = $4
		LIMIT 1
	`
	return r.scanLocation(r.db.QueryRowContext(ctx, query, wcdma.MCC, wcdma.MNC, wcdma.PSC, wcdma.CID))
}

func (r *LocateRepo) FindWifi(ctx context.Context, wifi *model.Wifi) (*model.Location, error) {
	const query = `
		SELECT lat, lon
		FROM wifi
		WHERE bssid = $1 AND lat IS NOT NULL AND lon IS NOT NULL
		LIMIT 1
	`
	row := r.db.QueryRowContext(ctx, query, wifi.BSSID)
	var loc model.Location
	if err := row.Scan(&loc.Point.Lat, &loc.Point.Lon); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("FindWifi scan: %w", err)
	}
	return &loc, nil
}

func (r *LocateRepo) FindIP(ctx context.Context, ip *model.Ip) (*model.Location, error) {
	const query = `
		SELECT lat, lon
		FROM ip
		WHERE address = $1 AND lat IS NOT NULL AND lon IS NOT NULL
		LIMIT 1
	`
	row := r.db.QueryRowContext(ctx, query, ip.Address)
	var loc model.Location
	if err := row.Scan(&loc.Point.Lat, &loc.Point.Lon); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("FindIP scan: %w", err)
	}
	return &loc, nil
}

func (r *LocateRepo) scanLocation(row *sql.Row) (*model.Location, error) {
	var loc model.Location
	if err := row.Scan(&loc.Point.Lat, &loc.Point.Lon, &loc.FromMetro); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("scan location: %w", err)
	}
	return &loc, nil
}

func (r *LocateRepo) UpsertCell(row model.CellRefreshRow) error {
	fromMetro := row.ObjectType.Valid && row.ObjectType.String == "metro"
	const query = `
		INSERT INTO cells (
			tech, mcc, mnc, lac, tac, cid, ci, psc, pci,
			arfcn, earfcn, uarfcn, bsic,
			gsm_timing_advance, lte_timing_advance,
			gsm_age, wcdma_age, lte_age,
			lat, lon, from_metro
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9,
			$10, $11, $12, $13, $14, $15, $16, $17, $18,
			$19, $20, $21
		)
		ON CONFLICT (tech, mcc, mnc, COALESCE(lac, tac), COALESCE(cid, ci))
		DO UPDATE SET lat = EXCLUDED.lat, lon = EXCLUDED.lon, from_metro = EXCLUDED.from_metro
	`
	_, err := r.db.Exec(query,
		row.Tech, row.MCC, row.MNC, row.LAC, row.TAC, row.CID, row.CI, row.PSC, row.PCI,
		row.ARFCN, row.EARFCN, row.UARFCN, row.BSIC,
		row.GSMTimingAdvance, row.LTETimingAdvance,
		row.GSMAge, row.WCDMAAge, row.LTEAge,
		row.Lat, row.Lon, fromMetro,
	)
	return err
}

func (r *LocateRepo) UpsertWifi(row model.WifiRefreshRow) error {
	const query = `
		INSERT INTO wifi (bssid, lat, lon)
		VALUES ($1, $2, $3)
		ON CONFLICT (bssid) DO UPDATE SET lat = EXCLUDED.lat, lon = EXCLUDED.lon
	`
	_, err := r.db.Exec(query, row.BSSID, row.Lat, row.Lon)
	return err
}

func (r *LocateRepo) UpsertIP(row model.IPRefreshRow) error {
	const query = `
		INSERT INTO ip (address, lat, lon)
		VALUES ($1, $2, $3)
		ON CONFLICT (address) DO UPDATE SET lat = EXCLUDED.lat, lon = EXCLUDED.lon
	`
	_, err := r.db.Exec(query, row.Address, row.Lat, row.Lon)
	return err
}
