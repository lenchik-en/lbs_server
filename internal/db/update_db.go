package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/lenchik-en/lbs_server/internal/api"
)

type UpdateDB struct {
	DB *sql.DB
}

func NewUpdateDB(dsn string) (*UpdateDB, error) {
	db, err := OpenPostgres(dsn)
	if err != nil {
		return nil, fmt.Errorf("locateDB: %w", err)
	}
	fmt.Println("Connection to UpdateDB is OK")
	return &UpdateDB{
		DB: db,
	}, nil
}

func (u *UpdateDB) InsertCell(ctx context.Context, cell *api.Cell, loc *api.Location, rawJSON any) error {
	data, err := json.Marshal(rawJSON)
	if err != nil {
		return err
	}

	switch {
	case cell.LTE != nil:
		query := `
		INSERT INTO cells
            (tech, mcc, mnc, tac, ci, lat, lon, source, raw)
        VALUES
            ('LTE', $1, $2, $3, $4, $5, $6, 'client', $7)
        ON CONFLICT DO NOTHING 
		`
		_, err := u.DB.ExecContext(ctx, query, cell.LTE.MCC, cell.LTE.MNC, cell.LTE.TAC, cell.LTE.CI, loc.Point.Lat, loc.Point.Lon, data)
		return err
	default:
		return fmt.Errorf("unsupported cell type")
	}
	return nil
}
