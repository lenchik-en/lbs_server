package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/lenchik-en/lbs_server/internal/models"
	_ "github.com/lib/pq"
)

func OpenPostgres(dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to the database: %v", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %v", err)
	}
	return db, nil
}

type LocateDB struct {
	DB *sql.DB
}

func NewLocateDB(dsn string) (*LocateDB, error) {
	db, err := OpenPostgres(dsn)
	if err != nil {
		return nil, fmt.Errorf("locateDB: %w", err)
	}
	fmt.Println("Connection to LocateDB is OK")
	return &LocateDB{
		DB: db,
	}, nil
}

func (l *LocateDB) GetConnection() *sql.DB { return l.DB }

type CellFinder interface {
	FindLTE(ctx context.Context, lte *models.LTE) (*models.Location, error)
	FindGSM(ctx context.Context, gsm *models.GSM) (*models.Location, error)
	FindWCDMA(ctx context.Context, wcdma *models.WCDMA) (*models.Location, error)
	GetConnection() *sql.DB
}

type WifiFinder interface {
	FindWifi(ctx context.Context, wifi *models.Wifi) (*models.Location, error)
}

type IPFinder interface {
	FindIP(ctx context.Context, ip *models.Ip) (*models.Location, error)
}

type Finder interface {
	CellFinder
	WifiFinder
	IPFinder
}

func (l *LocateDB) FindLTE(ctx context.Context, lte *models.LTE) (*models.Location, error) {
	const query = `
		SELECT lat, lon, from_metro
        FROM cells
        WHERE tech = 'LTE'
          AND mcc = $1
          AND mnc = $2
          AND tac = $3
          AND ci  = $4
        LIMIT 1
	`

	row := l.DB.QueryRowContext(ctx, query, lte.MCC, lte.MNC, lte.TAC, lte.CI)

	var loc models.Location
	if err := row.Scan(&loc.Point.Lat, &loc.Point.Lon, &loc.FromMetro); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("error while row.Scan: %v", err)
	}
	return &loc, nil
}

func (l *LocateDB) FindGSM(ctx context.Context, gse *models.GSM) (*models.Location, error) {
	const query = `
		SELECT lat, lon, from_metro
        FROM cells
        WHERE tech = 'GSM'
          AND mcc = $1
          AND mnc = $2
          AND lac = $3
          AND cid  = $4
        LIMIT 1
	`

	row := l.DB.QueryRowContext(ctx, query, gse.MCC, gse.MNC, gse.LAC, gse.CID)

	var loc models.Location
	if err := row.Scan(&loc.Point.Lat, &loc.Point.Lon, &loc.FromMetro); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("error while row.Scan: %v", err)
	}
	return &loc, nil
}

func (l *LocateDB) FindWCDMA(ctx context.Context, wcdma *models.WCDMA) (*models.Location, error) {
	const query = `
		SELECT lat, lon, from_metro
        FROM cells
        WHERE tech = 'WCDMA'
          AND mcc = $1
          AND mnc = $2
          AND psc = $3
          AND cid  = $4
        LIMIT 1
	`

	row := l.DB.QueryRowContext(ctx, query, wcdma.MCC, wcdma.MNC, wcdma.PSC, wcdma.CID)

	var loc models.Location
	if err := row.Scan(&loc.Point.Lat, &loc.Point.Lon, &loc.FromMetro); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("error while row.Scan: %v", err)
	}
	return &loc, nil
}

func (l *LocateDB) FindWifi(ctx context.Context, wifi *models.Wifi) (*models.Location, error) {
	const query = `
		SELECT lat, lon
		FROM wifi
		WHERE bssid = $1
		  AND lat IS NOT NULL
		  AND lon IS NOT NULL
		LIMIT 1
	`

	row := l.DB.QueryRowContext(ctx, query, wifi.BSSID)

	var loc models.Location
	if err := row.Scan(&loc.Point.Lat, &loc.Point.Lon); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("error while row.Scan: %v", err)
	}
	return &loc, nil
}

func (l *LocateDB) FindIP(ctx context.Context, ip *models.Ip) (*models.Location, error) {
	const query = `
		SELECT lat, lon
		FROM ip
		WHERE address = $1
		  AND lat IS NOT NULL
		  AND lon IS NOT NULL
		LIMIT 1
	`

	row := l.DB.QueryRowContext(ctx, query, ip.Address)

	var loc models.Location
	if err := row.Scan(&loc.Point.Lat, &loc.Point.Lon); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("error while row.Scan: %v", err)
	}
	return &loc, nil
}

func (l *LocateDB) UpsertCell(row CellRefreshRow) error {
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
			$10, $11, $12, $13,
			$14, $15,
			$16, $17, $18,
			$19, $20, $21
		)
		ON CONFLICT (tech, mcc, mnc, COALESCE(lac, tac), COALESCE(cid, ci))
		DO UPDATE SET lat = EXCLUDED.lat, lon = EXCLUDED.lon, from_metro = EXCLUDED.from_metro
	`
	_, err := l.DB.Exec(query,
		row.Tech, row.MCC, row.MNC, row.LAC, row.TAC, row.CID, row.CI, row.PSC, row.PCI,
		row.ARFCN, row.EARFCN, row.UARFCN, row.BSIC,
		row.GSMTimingAdvance, row.LTETimingAdvance,
		row.GSMAge, row.WCDMAAge, row.LTEAge,
		row.Lat, row.Lon, fromMetro,
	)
	return err
}

func (l *LocateDB) UpsertWifi(row WifiRefreshRow) error {
	const query = `
		INSERT INTO wifi (bssid, lat, lon)
		VALUES ($1, $2, $3)
		ON CONFLICT (bssid) DO UPDATE SET lat = EXCLUDED.lat, lon = EXCLUDED.lon
	`
	_, err := l.DB.Exec(query, row.BSSID, row.Lat, row.Lon)
	return err
}

func (l *LocateDB) UpsertIP(row IPRefreshRow) error {
	const query = `
		INSERT INTO ip (address, lat, lon)
		VALUES ($1, $2, $3)
		ON CONFLICT (address) DO UPDATE SET lat = EXCLUDED.lat, lon = EXCLUDED.lon
	`
	_, err := l.DB.Exec(query, row.Address, row.Lat, row.Lon)
	return err
}
