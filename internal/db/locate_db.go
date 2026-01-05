package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/lenchik-en/lbs_server/internal/api"
	_ "github.com/lib/pq"
)

func OpenPostgres(dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to the database: %v", err)
	}

	//TODO: is it necessary?
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
	//TODO: убрать зависимость от api
	FindLTE(ctx context.Context, lte *api.LTE) (*api.Location, error)
	FindGSM(ctx context.Context, gse *api.GSM) (*api.Location, error)
	FindWCDMA(ctx context.Context, wcdma *api.WCDMA) (*api.Location, error)
}

func (l *LocateDB) FindLTE(ctx context.Context, lte *api.LTE) (*api.Location, error) {
	const query = `
		SELECT lat, lon
        FROM cells
        WHERE tech = 'LTE'
          AND mcc = $1
          AND mnc = $2
          AND tac = $3
          AND ci  = $4
        LIMIT 1
	`

	row := l.DB.QueryRowContext(ctx, query, lte.MCC, lte.MNC, lte.TAC, lte.CI)

	var loc api.Location
	if err := row.Scan(&loc.Point.Lat, &loc.Point.Lon); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			fmt.Printf("Query has no rows")
			return nil, nil
		}

		return nil, fmt.Errorf("error while row.Scan: %v", err)
	}
	//TODO:accuracy?
	loc.Accuracy = 500

	return &loc, nil
}

func (l *LocateDB) FindGSM(ctx context.Context, gse *api.GSM) (*api.Location, error) {
	const query = `
		SELECT lat, lon
        FROM cells
        WHERE tech = 'GSM'
          AND mcc = $1
          AND mnc = $2
          AND lac = $3
          AND cid  = $4
        LIMIT 1
	`

	row := l.DB.QueryRowContext(ctx, query, gse.MCC, gse.MNC, gse.LAC, gse.CID)

	var loc api.Location
	if err := row.Scan(&loc.Point.Lat, &loc.Point.Lon); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			fmt.Printf("Query has no rows")
			return nil, nil
		}

		return nil, fmt.Errorf("error while row.Scan: %v", err)
	}
	//TODO:accuracy?
	loc.Accuracy = 500

	return &loc, nil
}

func (l *LocateDB) FindWCDMA(ctx context.Context, wcdma *api.WCDMA) (*api.Location, error) {
	const query = `
		SELECT lat, lon
        FROM cells
        WHERE tech = 'WCDMA'
          AND mcc = $1
          AND mnc = $2
          AND psc = $3
          AND cid  = $4
        LIMIT 1
	`

	row := l.DB.QueryRowContext(ctx, query, wcdma.MCC, wcdma.MNC, wcdma.PSC, wcdma.CID)

	var loc api.Location
	if err := row.Scan(&loc.Point.Lat, &loc.Point.Lon); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			fmt.Printf("Query has no rows")
			return nil, nil
		}

		return nil, fmt.Errorf("error while row.Scan: %v", err)
	}
	//TODO:accuracy?
	loc.Accuracy = 500

	return &loc, nil
}
