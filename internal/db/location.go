package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/lenchik-en/lbs_server/internal/api"
)

type Location struct {
	Point struct {
		Lat float64
		Lon float64
	}
	Accuracy int
	//TODO: добавить источник? locate/external
}

type LocateCellStore interface {
	//TODO: убрать зависимость от api.LTE
	FindLTE(ctx context.Context, lte *api.LTE) (*Location, error)
	//FindGSM
	//FindWCDMA
}

func (l *LocateDB) FindLTE(ctx context.Context, lte *api.LTE) (*Location, error) {
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

	var loc Location
	if err := row.Scan(&loc.Point.Lat, &loc.Point.Lon); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			fmt.Printf("Query has no rows")
			return nil, nil
		}

		return nil, fmt.Errorf("error while row.Scan: %v", err)
	}
	//TODO:?
	loc.Accuracy = 500

	return &loc, nil
}
