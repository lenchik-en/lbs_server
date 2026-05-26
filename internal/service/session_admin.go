package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/lenchik-en/lbs_server/internal/domain/model"
	"github.com/lenchik-en/lbs_server/internal/domain/port"
)

const SessionPageSize = 50

type SessionAdminService struct {
	repo port.SessionRepository
	dest port.RefreshDest
}

func NewSessionAdminService(repo port.SessionRepository, dest port.RefreshDest) *SessionAdminService {
	return &SessionAdminService{repo: repo, dest: dest}
}

func (s *SessionAdminService) List(ctx context.Context, page int) ([]model.SessionListRow, int64, error) {
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * SessionPageSize
	rows, total, err := s.repo.ListSessions(ctx, SessionPageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("session list: %w", err)
	}
	return rows, total, nil
}

func (s *SessionAdminService) GetPoints(ctx context.Context, sessionUUID string) ([]model.SessionPoint, error) {
	points, err := s.repo.GetSessionPoints(ctx, sessionUUID)
	if err != nil {
		return nil, fmt.Errorf("session points: %w", err)
	}
	return points, nil
}

func (s *SessionAdminService) UpdatePoint(ctx context.Context, id int64, lat, lon float64, accuracy float64) error {
	return s.repo.UpdatePoint(ctx, id, lat, lon, accuracy)
}

// RefreshFromSession перебирает точки сессии, разбирает raw_request каждой точки
// и делает upsert ячеек/wifi/IP в LocateDB с (возможно скорректированными) координатами.
func (s *SessionAdminService) RefreshFromSession(ctx context.Context, sessionUUID string) (model.RefreshStats, error) {
	points, err := s.repo.GetSessionPoints(ctx, sessionUUID)
	if err != nil {
		return model.RefreshStats{}, fmt.Errorf("get points: %w", err)
	}

	var stats model.RefreshStats
	for _, p := range points {
		if len(p.RawRequest) == 0 {
			continue
		}
		var req struct {
			Cell []model.Cell `json:"cell"`
			Wifi []model.Wifi `json:"wifi"`
			IP   []model.Ip   `json:"ip"`
		}
		if err := json.Unmarshal(p.RawRequest, &req); err != nil {
			continue
		}
		for _, c := range req.Cell {
			row, ok := cellToRefreshRow(c, p.Lat, p.Lon)
			if !ok {
				continue
			}
			if err := s.dest.UpsertCell(row); err == nil {
				stats.CellsUpserted++
			}
		}
		for _, w := range req.Wifi {
			if w.BSSID == "" {
				continue
			}
			if err := s.dest.UpsertWifi(model.WifiRefreshRow{BSSID: w.BSSID, Lat: p.Lat, Lon: p.Lon}); err == nil {
				stats.WifisUpserted++
			}
		}
		for _, ip := range req.IP {
			if ip.Address == "" {
				continue
			}
			if err := s.dest.UpsertIP(model.IPRefreshRow{Address: ip.Address, Lat: p.Lat, Lon: p.Lon}); err == nil {
				stats.IPsUpserted++
			}
		}
	}
	return stats, nil
}

func cellToRefreshRow(c model.Cell, lat, lon float64) (model.CellRefreshRow, bool) {
	row := model.CellRefreshRow{Lat: lat, Lon: lon}
	switch {
	case c.LTE != nil:
		row.Tech = "lte"
		row.MCC = int32(c.LTE.MCC)
		row.MNC = int32(c.LTE.MNC)
		row.TAC = sql.NullInt32{Int32: int32(c.LTE.TAC), Valid: true}
		row.CI = sql.NullInt32{Int32: int32(c.LTE.CI), Valid: true}
		row.PCI = sql.NullInt32{Int32: int32(c.LTE.PCI), Valid: true}
		row.EARFCN = sql.NullInt32{Int32: int32(c.LTE.EARFCN), Valid: true}
		row.LTETimingAdvance = sql.NullInt32{Int32: int32(c.LTE.TimingAdvance), Valid: true}
		row.LTEAge = sql.NullInt32{Int32: int32(c.LTE.Age), Valid: true}
	case c.GSM != nil:
		row.Tech = "gsm"
		row.MCC = int32(c.GSM.MCC)
		row.MNC = int32(c.GSM.MNC)
		row.LAC = sql.NullInt32{Int32: int32(c.GSM.LAC), Valid: true}
		row.CID = sql.NullInt32{Int32: int32(c.GSM.CID), Valid: true}
		row.BSIC = sql.NullInt32{Int32: int32(c.GSM.BSIC), Valid: true}
		row.ARFCN = sql.NullInt32{Int32: int32(c.GSM.ARFCN), Valid: true}
		row.GSMTimingAdvance = sql.NullInt32{Int32: int32(c.GSM.TimingAdvance), Valid: true}
		row.GSMAge = sql.NullInt32{Int32: int32(c.GSM.Age), Valid: true}
	case c.WCDMA != nil:
		row.Tech = "wcdma"
		row.MCC = int32(c.WCDMA.MCC)
		row.MNC = int32(c.WCDMA.MNC)
		row.LAC = sql.NullInt32{Int32: int32(c.WCDMA.LAC), Valid: true}
		row.CID = sql.NullInt32{Int32: int32(c.WCDMA.CID), Valid: true}
		row.PSC = sql.NullInt32{Int32: int32(c.WCDMA.PSC), Valid: true}
		row.UARFCN = sql.NullInt32{Int32: int32(c.WCDMA.UARFCN), Valid: true}
		row.WCDMAAge = sql.NullInt32{Int32: int32(c.WCDMA.Age), Valid: true}
	default:
		return row, false
	}
	return row, true
}
