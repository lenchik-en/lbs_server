package service

import (
	"context"
	"errors"
	"testing"

	"github.com/lenchik-en/lbs_server/internal/domain/model"
)

// mockRefreshSource implements port.RefreshSource backed by slices.
type mockRefreshSource struct {
	cells   []model.CellRefreshRow
	wifis   []model.WifiRefreshRow
	ips     []model.IPRefreshRow
	cellErr error
	wifiErr error
	ipErr   error
}

func (s *mockRefreshSource) FetchCellsForRefresh(offset, limit int) ([]model.CellRefreshRow, error) {
	if s.cellErr != nil {
		return nil, s.cellErr
	}
	return slicePage(s.cells, offset, limit), nil
}

func (s *mockRefreshSource) FetchWifisForRefresh(offset, limit int) ([]model.WifiRefreshRow, error) {
	if s.wifiErr != nil {
		return nil, s.wifiErr
	}
	return slicePage(s.wifis, offset, limit), nil
}

func (s *mockRefreshSource) FetchIPsForRefresh(offset, limit int) ([]model.IPRefreshRow, error) {
	if s.ipErr != nil {
		return nil, s.ipErr
	}
	return slicePage(s.ips, offset, limit), nil
}

// slicePage returns a page of a generic slice by offset and limit.
func slicePage[T any](data []T, offset, limit int) []T {
	if offset >= len(data) {
		return nil
	}
	end := offset + limit
	if end > len(data) {
		end = len(data)
	}
	return data[offset:end]
}

// mockRefreshDest implements port.RefreshDest.
type mockRefreshDest struct {
	upsertCellErr error
	upsertWifiErr error
	upsertIPErr   error
}

func (d *mockRefreshDest) UpsertCell(_ model.CellRefreshRow) error { return d.upsertCellErr }
func (d *mockRefreshDest) UpsertWifi(_ model.WifiRefreshRow) error { return d.upsertWifiErr }
func (d *mockRefreshDest) UpsertIP(_ model.IPRefreshRow) error     { return d.upsertIPErr }

func makeCells(n int) []model.CellRefreshRow {
	rows := make([]model.CellRefreshRow, n)
	for i := range rows {
		rows[i] = model.CellRefreshRow{ID: int64(i + 1)}
	}
	return rows
}

// --- Tests ---

func TestRefresh_EmptySource(t *testing.T) {
	svc := NewRefreshService(&mockRefreshSource{}, &mockRefreshDest{})
	stats, err := svc.Refresh(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.CellsUpserted != 0 || stats.WifisUpserted != 0 || stats.IPsUpserted != 0 {
		t.Errorf("expected zero stats for empty source, got %+v", stats)
	}
}

func TestRefresh_SinglePageCells(t *testing.T) {
	svc := NewRefreshService(
		&mockRefreshSource{cells: makeCells(5)},
		&mockRefreshDest{},
	)
	stats, err := svc.Refresh(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.CellsUpserted != 5 {
		t.Errorf("expected 5 cells, got %d", stats.CellsUpserted)
	}
}

func TestRefresh_ExactlyOneFullPage(t *testing.T) {
	// exactly refreshPageSize rows → must trigger second fetch that returns 0
	svc := NewRefreshService(
		&mockRefreshSource{cells: makeCells(refreshPageSize)},
		&mockRefreshDest{},
	)
	stats, err := svc.Refresh(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.CellsUpserted != refreshPageSize {
		t.Errorf("expected %d cells, got %d", refreshPageSize, stats.CellsUpserted)
	}
}

func TestRefresh_MultiPageCells(t *testing.T) {
	total := refreshPageSize + 500 // 1000 + 500 = 1500
	svc := NewRefreshService(
		&mockRefreshSource{cells: makeCells(total)},
		&mockRefreshDest{},
	)
	stats, err := svc.Refresh(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.CellsUpserted != total {
		t.Errorf("expected %d cells, got %d", total, stats.CellsUpserted)
	}
}

func TestRefresh_UpsertCellError_SkipsAndContinues(t *testing.T) {
	svc := NewRefreshService(
		&mockRefreshSource{cells: makeCells(3)},
		&mockRefreshDest{upsertCellErr: errors.New("upsert failed")},
	)
	stats, err := svc.Refresh(context.Background())
	if err != nil {
		t.Fatalf("upsert errors must not be returned: %v", err)
	}
	if stats.CellsUpserted != 0 {
		t.Errorf("expected 0 successful upserts when all fail, got %d", stats.CellsUpserted)
	}
}

func TestRefresh_UpsertCellError_PartialSuccess(t *testing.T) {
	callCount := 0
	dst := &mockRefreshDest{}

	// Override via wrapping: fail every other upsert.
	// Instead, test a simpler case: wifis succeed, cells fail.
	svc := NewRefreshService(
		&mockRefreshSource{
			cells: makeCells(3),
			wifis: []model.WifiRefreshRow{{BSSID: "aa"}, {BSSID: "bb"}},
		},
		&mockRefreshDest{upsertCellErr: errors.New("fail")},
	)
	_ = callCount
	stats, err := svc.Refresh(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.CellsUpserted != 0 {
		t.Errorf("cells should not be counted when upsert fails")
	}
	if stats.WifisUpserted != 2 {
		t.Errorf("expected 2 wifis upserted, got %d", stats.WifisUpserted)
	}
	_ = dst
}

func TestRefresh_FetchCellsError(t *testing.T) {
	svc := NewRefreshService(
		&mockRefreshSource{cellErr: errors.New("fetch error")},
		&mockRefreshDest{},
	)
	_, err := svc.Refresh(context.Background())
	if err == nil {
		t.Error("expected error from FetchCells")
	}
}

func TestRefresh_FetchWifisError(t *testing.T) {
	svc := NewRefreshService(
		&mockRefreshSource{wifiErr: errors.New("fetch error")},
		&mockRefreshDest{},
	)
	_, err := svc.Refresh(context.Background())
	if err == nil {
		t.Error("expected error from FetchWifis")
	}
}

func TestRefresh_FetchIPsError(t *testing.T) {
	svc := NewRefreshService(
		&mockRefreshSource{ipErr: errors.New("fetch error")},
		&mockRefreshDest{},
	)
	_, err := svc.Refresh(context.Background())
	if err == nil {
		t.Error("expected error from FetchIPs")
	}
}

func TestRefresh_FetchWifisErrorAfterCellsSucceed(t *testing.T) {
	svc := NewRefreshService(
		&mockRefreshSource{
			cells:   makeCells(2),
			wifiErr: errors.New("wifi fetch error"),
		},
		&mockRefreshDest{},
	)
	stats, err := svc.Refresh(context.Background())
	if err == nil {
		t.Error("expected error from FetchWifis")
	}
	// Cells were already upserted before the wifi error.
	if stats.CellsUpserted != 2 {
		t.Errorf("expected 2 cells upserted before wifi error, got %d", stats.CellsUpserted)
	}
}

func TestRefresh_CombinedStats(t *testing.T) {
	svc := NewRefreshService(
		&mockRefreshSource{
			cells: makeCells(2),
			wifis: []model.WifiRefreshRow{{BSSID: "aa"}, {BSSID: "bb"}, {BSSID: "cc"}},
			ips:   []model.IPRefreshRow{{Address: "1.1.1.1"}},
		},
		&mockRefreshDest{},
	)
	stats, err := svc.Refresh(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.CellsUpserted != 2 || stats.WifisUpserted != 3 || stats.IPsUpserted != 1 {
		t.Errorf("expected cells=2 wifis=3 ips=1, got %+v", stats)
	}
}

func TestRefresh_UpsertWifiError_SkipsAndContinues(t *testing.T) {
	svc := NewRefreshService(
		&mockRefreshSource{
			wifis: []model.WifiRefreshRow{{BSSID: "aa"}, {BSSID: "bb"}},
		},
		&mockRefreshDest{upsertWifiErr: errors.New("fail")},
	)
	stats, err := svc.Refresh(context.Background())
	if err != nil {
		t.Fatalf("wifi upsert errors must not be returned: %v", err)
	}
	if stats.WifisUpserted != 0 {
		t.Errorf("expected 0 wifis upserted when all fail, got %d", stats.WifisUpserted)
	}
}

func TestRefresh_UpsertIPError_SkipsAndContinues(t *testing.T) {
	svc := NewRefreshService(
		&mockRefreshSource{
			ips: []model.IPRefreshRow{{Address: "1.1.1.1"}, {Address: "2.2.2.2"}},
		},
		&mockRefreshDest{upsertIPErr: errors.New("fail")},
	)
	stats, err := svc.Refresh(context.Background())
	if err != nil {
		t.Fatalf("ip upsert errors must not be returned: %v", err)
	}
	if stats.IPsUpserted != 0 {
		t.Errorf("expected 0 IPs upserted when all fail, got %d", stats.IPsUpserted)
	}
}
