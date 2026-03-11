package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/lenchik-en/lbs_server/internal/models"
)

type ExternalDB struct {
	DB *sql.DB
}

func NewExternalDB(dsn string) (*ExternalDB, error) {
	db, err := OpenPostgres(dsn)
	if err != nil {
		return nil, fmt.Errorf("externalDB: %w", err)
	}
	fmt.Println("Connection to the ExternalDB is OK")
	return &ExternalDB{DB: db}, nil
}

func (e *ExternalDB) FindLTE(ctx context.Context, lte *models.LTE) (*models.Location, error) {
	const query = `
			SELECT 
			    lat, 
			    lon
			FROM cells
			WHERE 
			    radio = 3
				AND mcc = $1
				AND mnc = $2
				AND area = $3
				AND cell = $4
			LIMIT 1      
	`

	row := e.DB.QueryRowContext(ctx, query, lte.MCC, lte.MNC, lte.TAC, lte.CI)

	var loc models.Location
	if err := row.Scan(&loc.Point.Lat, &loc.Point.Lon); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			fmt.Printf("Query has no rows")
			return nil, nil
		}

		return nil, fmt.Errorf("error while row.Scan: %v", err)
	}

	loc.Accuracy = 500
	return &loc, nil
}

func (e *ExternalDB) FindGSM(ctx context.Context, gse *models.GSM) (*models.Location, error) {
	const query = `
		SELECT lat, lon
        FROM cells
        WHERE radio = 1
          AND mcc = $1
          AND mnc = $2
          AND area = $3
          AND cell  = $4
        LIMIT 1
	`

	row := e.DB.QueryRowContext(ctx, query, gse.MCC, gse.MNC, gse.LAC, gse.CID)

	var loc models.Location
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

func (e *ExternalDB) FindWCDMA(ctx context.Context, wcdma *models.WCDMA) (*models.Location, error) {
	const query = `
		SELECT lat, lon
        FROM cells
        WHERE radio = 2
          AND mcc = $1
          AND mnc = $2
          AND area = $3
          AND cell  = $4
        LIMIT 1
	`

	row := e.DB.QueryRowContext(ctx, query, wcdma.MCC, wcdma.MNC, wcdma.LAC, wcdma.CID)

	var loc models.Location
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

func (e *ExternalDB) GetConnection() *sql.DB {
	return e.DB
}
