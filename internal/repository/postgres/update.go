package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/lenchik-en/lbs_server/internal/domain/model"
)

type UpdateRepo struct {
	db *sql.DB
}

func NewUpdateRepo(dsn string) (*UpdateRepo, error) {
	db, err := Open(dsn)
	if err != nil {
		return nil, fmt.Errorf("updateRepo: %w", err)
	}
	return &UpdateRepo{db: db}, nil
}

func (r *UpdateRepo) DB() *sql.DB { return r.db }

func (r *UpdateRepo) InsertCell(ctx context.Context, cell *model.Cell, loc *model.Location, source, objectType string, rawJSON any) error {
	data, err := json.Marshal(rawJSON)
	if err != nil {
		return err
	}
	switch {
	case cell.LTE != nil:
		const query = `
			INSERT INTO cells (tech, mcc, mnc, tac, ci, pci, earfcn, lte_timing_advance, lte_age, lat, lon, source, type, raw)
			VALUES ('LTE', $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
			ON CONFLICT DO NOTHING
		`
		_, err = r.db.ExecContext(ctx, query,
			cell.LTE.MCC, cell.LTE.MNC, cell.LTE.TAC, cell.LTE.CI,
			cell.LTE.PCI, cell.LTE.EARFCN, cell.LTE.TimingAdvance, cell.LTE.Age,
			loc.Point.Lat, loc.Point.Lon, source, objectType, data)
	case cell.GSM != nil:
		const query = `
			INSERT INTO cells (tech, mcc, mnc, lac, cid, bsic, arfcn, gsm_timing_advance, gsm_age, lat, lon, source, type, raw)
			VALUES ('GSM', $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
			ON CONFLICT DO NOTHING
		`
		_, err = r.db.ExecContext(ctx, query,
			cell.GSM.MCC, cell.GSM.MNC, cell.GSM.LAC, cell.GSM.CID,
			cell.GSM.BSIC, cell.GSM.ARFCN, cell.GSM.TimingAdvance, cell.GSM.Age,
			loc.Point.Lat, loc.Point.Lon, source, objectType, data)
	case cell.WCDMA != nil:
		const query = `
			INSERT INTO cells (tech, mcc, mnc, lac, cid, psc, uarfcn, wcdma_age, lat, lon, source, type, raw)
			VALUES ('WCDMA', $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
			ON CONFLICT DO NOTHING
		`
		_, err = r.db.ExecContext(ctx, query,
			cell.WCDMA.MCC, cell.WCDMA.MNC, cell.WCDMA.LAC, cell.WCDMA.CID,
			cell.WCDMA.PSC, cell.WCDMA.UARFCN, cell.WCDMA.Age,
			loc.Point.Lat, loc.Point.Lon, source, objectType, data)
	default:
		return fmt.Errorf("unsupported cell type")
	}
	return err
}

func (r *UpdateRepo) InsertWifi(ctx context.Context, wifi *model.Wifi, loc *model.Location, source, objectType string, rawJSON any) error {
	data, err := json.Marshal(rawJSON)
	if err != nil {
		return err
	}
	const query = `
		INSERT INTO wifi (bssid, signal_strength, channel, age, lat, lon, source, type, raw)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err = r.db.ExecContext(ctx, query,
		wifi.BSSID, wifi.SignalStrength, wifi.CHANNEL, wifi.AGE,
		loc.Point.Lat, loc.Point.Lon, source, objectType, data)
	return err
}

func (r *UpdateRepo) InsertIP(ctx context.Context, ip *model.Ip, loc *model.Location, source, objectType string, rawJSON any) error {
	data, err := json.Marshal(rawJSON)
	if err != nil {
		return err
	}
	const query = `
		INSERT INTO ip (address, lat, lon, source, type, raw)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT DO NOTHING
	`
	_, err = r.db.ExecContext(ctx, query,
		ip.Address, loc.Point.Lat, loc.Point.Lon, source, objectType, data)
	return err
}

const refreshPageSize = 1000

func (r *UpdateRepo) FetchCellsForRefresh(offset, limit int) ([]model.CellRefreshRow, error) {
	const query = `
		SELECT id, tech, mcc, mnc,
		       lac, tac, cid, ci, psc, pci, arfcn, earfcn, uarfcn, bsic,
		       gsm_timing_advance, lte_timing_advance, gsm_age, wcdma_age, lte_age,
		       lat, lon, type
		FROM cells
		WHERE lat IS NOT NULL AND lon IS NOT NULL
		ORDER BY id LIMIT $1 OFFSET $2
	`
	rows, err := r.db.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []model.CellRefreshRow
	for rows.Next() {
		var row model.CellRefreshRow
		if err := rows.Scan(
			&row.ID, &row.Tech, &row.MCC, &row.MNC,
			&row.LAC, &row.TAC, &row.CID, &row.CI, &row.PSC, &row.PCI,
			&row.ARFCN, &row.EARFCN, &row.UARFCN, &row.BSIC,
			&row.GSMTimingAdvance, &row.LTETimingAdvance,
			&row.GSMAge, &row.WCDMAAge, &row.LTEAge,
			&row.Lat, &row.Lon, &row.ObjectType,
		); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

func (r *UpdateRepo) FetchWifisForRefresh(offset, limit int) ([]model.WifiRefreshRow, error) {
	const query = `
		SELECT bssid, lat, lon FROM wifi
		WHERE lat IS NOT NULL AND lon IS NOT NULL
		ORDER BY id LIMIT $1 OFFSET $2
	`
	rows, err := r.db.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []model.WifiRefreshRow
	for rows.Next() {
		var row model.WifiRefreshRow
		if err := rows.Scan(&row.BSSID, &row.Lat, &row.Lon); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

func (r *UpdateRepo) FetchIPsForRefresh(offset, limit int) ([]model.IPRefreshRow, error) {
	const query = `
		SELECT address, lat, lon FROM ip
		WHERE lat IS NOT NULL AND lon IS NOT NULL
		ORDER BY id LIMIT $1 OFFSET $2
	`
	rows, err := r.db.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []model.IPRefreshRow
	for rows.Next() {
		var row model.IPRefreshRow
		if err := rows.Scan(&row.Address, &row.Lat, &row.Lon); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}
