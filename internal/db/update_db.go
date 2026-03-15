package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/lenchik-en/lbs_server/internal/models"
)

type Inserter interface {
	InsertCell(ctx context.Context, cell *models.Cell, loc *models.Location, source string, rawJSON any) error
	InsertWifi(ctx context.Context, wifi *models.Wifi, source string, rawJSON any) error
	InsertIP(ctx context.Context, ip *models.Ip, source string, rawJSON any) error
	GetConnection() *sql.DB
}

type UpdateDB struct {
	DB *sql.DB
}

func NewUpdateDB(dsn string) (*UpdateDB, error) {
	db, err := OpenPostgres(dsn)
	if err != nil {
		return nil, fmt.Errorf("updateDB: %w", err)
	}
	fmt.Println("Connection to UpdateDB is OK")
	return &UpdateDB{
		DB: db,
	}, nil
}

func (u *UpdateDB) InsertCell(ctx context.Context, cell *models.Cell, loc *models.Location, source string, rawJSON any) error {
	data, err := json.Marshal(rawJSON)
	if err != nil {
		return err
	}

	switch {
	case cell.LTE != nil:
		query := `
		INSERT INTO cells
            (tech, mcc, mnc, tac, ci, pci, earfcn, lte_timing_advance, lte_age, lat, lon, source, raw)
        VALUES
            ('LTE', $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
        ON CONFLICT DO NOTHING
		`
		_, err := u.DB.ExecContext(ctx, query, cell.LTE.MCC, cell.LTE.MNC, cell.LTE.TAC, cell.LTE.CI, cell.LTE.PCI, cell.LTE.EARFCN, cell.LTE.TimingAdvance, cell.LTE.Age, loc.Point.Lat, loc.Point.Lon, source, data)
		return err
	case cell.GSM != nil:
		query := `
		INSERT INTO cells
            (tech, mcc, mnc, lac, cid, bsic, arfcn, gsm_timing_advance, gsm_age, lat, lon, source, raw)
        VALUES
            ('GSM', $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
        ON CONFLICT DO NOTHING
		`
		_, err := u.DB.ExecContext(ctx, query, cell.GSM.MCC, cell.GSM.MNC, cell.GSM.LAC, cell.GSM.CID, cell.GSM.BSIC, cell.GSM.ARFCN, cell.GSM.TimingAdvance, cell.GSM.Age, loc.Point.Lat, loc.Point.Lon, source, data)
		return err
	case cell.WCDMA != nil:
		query := `
		INSERT INTO cells
            (tech, mcc, mnc, lac, cid, psc, uarfcn, wcdma_age, lat, lon, source, raw)
        VALUES
            ('WCDMA', $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
        ON CONFLICT DO NOTHING
		`
		_, err := u.DB.ExecContext(ctx, query, cell.WCDMA.MCC, cell.WCDMA.MNC, cell.WCDMA.LAC, cell.WCDMA.CID, cell.WCDMA.PSC, cell.WCDMA.UARFCN, cell.WCDMA.Age, loc.Point.Lat, loc.Point.Lon, source, data)
		return err
	default:
		return fmt.Errorf("unsupported cell type")
	}
}

func (u *UpdateDB) InsertWifi(ctx context.Context, wifi *models.Wifi, source string, rawJSON any) error {
	data, err := json.Marshal(rawJSON)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO wifi (bssid, signal_strength, channel, age, source, raw)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err = u.DB.ExecContext(ctx, query, wifi.BSSID, wifi.SignalStrength, wifi.CHANNEL, wifi.AGE, source, data)
	return err
}

func (u *UpdateDB) InsertIP(ctx context.Context, ip *models.Ip, source string, rawJSON any) error {
	data, err := json.Marshal(rawJSON)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO ip (address, source, raw)
		VALUES ($1, $2, $3)
	`
	_, err = u.DB.ExecContext(ctx, query, ip.Address, source, data)
	return err
}

func (u *UpdateDB) GetConnection() *sql.DB {
	return u.DB
}
