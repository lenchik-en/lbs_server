package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/lenchik-en/lbs_server/internal/api"
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

type ExternalCellStore interface {
	FindLTE(ctx context.Context, lte *api.LTE) (*Location, error)
	//FindGSM
	//FindWCDMA
}

func (e *ExternalDB) FindLTE(ctx context.Context, lte *api.LTE) (*Location, error) {
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

	var loc Location
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
