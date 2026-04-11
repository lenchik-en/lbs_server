package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/lenchik-en/lbs_server/internal/models"
)

type Inserter interface {
	InsertCell(ctx context.Context, cell *models.Cell, loc *models.Location, source, objectType string, rawJSON any) error
	InsertWifi(ctx context.Context, wifi *models.Wifi, loc *models.Location, source, objectType string, rawJSON any) error
	InsertIP(ctx context.Context, ip *models.Ip, loc *models.Location, source, objectType string, rawJSON any) error
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

func (u *UpdateDB) InsertCell(ctx context.Context, cell *models.Cell, loc *models.Location, source, objectType string, rawJSON any) error {
	data, err := json.Marshal(rawJSON)
	if err != nil {
		return err
	}

	switch {
	case cell.LTE != nil:
		query := `
		INSERT INTO cells
            (tech, mcc, mnc, tac, ci, pci, earfcn, lte_timing_advance, lte_age, lat, lon, source, type, raw)
        VALUES
            ('LTE', $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
        ON CONFLICT DO NOTHING
		`
		_, err := u.DB.ExecContext(ctx, query,
			cell.LTE.MCC, cell.LTE.MNC, cell.LTE.TAC, cell.LTE.CI,
			cell.LTE.PCI, cell.LTE.EARFCN, cell.LTE.TimingAdvance, cell.LTE.Age,
			loc.Point.Lat, loc.Point.Lon, source, objectType, data)
		return err
	case cell.GSM != nil:
		query := `
		INSERT INTO cells
            (tech, mcc, mnc, lac, cid, bsic, arfcn, gsm_timing_advance, gsm_age, lat, lon, source, type, raw)
        VALUES
            ('GSM', $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
        ON CONFLICT DO NOTHING
		`
		_, err := u.DB.ExecContext(ctx, query,
			cell.GSM.MCC, cell.GSM.MNC, cell.GSM.LAC, cell.GSM.CID,
			cell.GSM.BSIC, cell.GSM.ARFCN, cell.GSM.TimingAdvance, cell.GSM.Age,
			loc.Point.Lat, loc.Point.Lon, source, objectType, data)
		return err
	case cell.WCDMA != nil:
		query := `
		INSERT INTO cells
            (tech, mcc, mnc, lac, cid, psc, uarfcn, wcdma_age, lat, lon, source, type, raw)
        VALUES
            ('WCDMA', $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
        ON CONFLICT DO NOTHING
		`
		_, err := u.DB.ExecContext(ctx, query,
			cell.WCDMA.MCC, cell.WCDMA.MNC, cell.WCDMA.LAC, cell.WCDMA.CID,
			cell.WCDMA.PSC, cell.WCDMA.UARFCN, cell.WCDMA.Age,
			loc.Point.Lat, loc.Point.Lon, source, objectType, data)
		return err
	default:
		return fmt.Errorf("unsupported cell type")
	}
}

func (u *UpdateDB) InsertWifi(ctx context.Context, wifi *models.Wifi, loc *models.Location, source, objectType string, rawJSON any) error {
	data, err := json.Marshal(rawJSON)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO wifi (bssid, signal_strength, channel, age, lat, lon, source, type, raw)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err = u.DB.ExecContext(ctx, query,
		wifi.BSSID, wifi.SignalStrength, wifi.CHANNEL, wifi.AGE,
		loc.Point.Lat, loc.Point.Lon, source, objectType, data)
	return err
}

func (u *UpdateDB) InsertIP(ctx context.Context, ip *models.Ip, loc *models.Location, source, objectType string, rawJSON any) error {
	data, err := json.Marshal(rawJSON)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO ip (address, lat, lon, source, type, raw)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT DO NOTHING
	`
	_, err = u.DB.ExecContext(ctx, query,
		ip.Address, loc.Point.Lat, loc.Point.Lon, source, objectType, data)
	return err
}

func (u *UpdateDB) GetConnection() *sql.DB {
	return u.DB
}

const refreshBatchSize = 1000

func (u *UpdateDB) FetchCellsForRefresh(offset, limit int) ([]CellRefreshRow, error) {
	const query = `
		SELECT id, tech, mcc, mnc,
		       lac, tac, cid, ci, psc, pci, arfcn, earfcn, uarfcn, bsic,
		       gsm_timing_advance, lte_timing_advance, gsm_age, wcdma_age, lte_age,
		       lat, lon, type
		FROM cells
		WHERE lat IS NOT NULL AND lon IS NOT NULL
		ORDER BY id
		LIMIT $1 OFFSET $2
	`
	rows, err := u.DB.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []CellRefreshRow
	for rows.Next() {
		var r CellRefreshRow
		if err := rows.Scan(
			&r.ID, &r.Tech, &r.MCC, &r.MNC,
			&r.LAC, &r.TAC, &r.CID, &r.CI, &r.PSC, &r.PCI,
			&r.ARFCN, &r.EARFCN, &r.UARFCN, &r.BSIC,
			&r.GSMTimingAdvance, &r.LTETimingAdvance,
			&r.GSMAge, &r.WCDMAAge, &r.LTEAge,
			&r.Lat, &r.Lon, &r.ObjectType,
		); err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

func (u *UpdateDB) FetchWifisForRefresh(offset, limit int) ([]WifiRefreshRow, error) {
	const query = `
		SELECT bssid, lat, lon
		FROM wifi
		WHERE lat IS NOT NULL AND lon IS NOT NULL
		ORDER BY id
		LIMIT $1 OFFSET $2
	`
	rows, err := u.DB.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []WifiRefreshRow
	for rows.Next() {
		var r WifiRefreshRow
		if err := rows.Scan(&r.BSSID, &r.Lat, &r.Lon); err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

func (u *UpdateDB) FetchIPsForRefresh(offset, limit int) ([]IPRefreshRow, error) {
	const query = `
		SELECT address, lat, lon
		FROM ip
		WHERE lat IS NOT NULL AND lon IS NOT NULL
		ORDER BY id
		LIMIT $1 OFFSET $2
	`
	rows, err := u.DB.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []IPRefreshRow
	for rows.Next() {
		var r IPRefreshRow
		if err := rows.Scan(&r.Address, &r.Lat, &r.Lon); err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}
