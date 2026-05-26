package service

import (
	"context"
	"testing"

	"github.com/lenchik-en/lbs_server/internal/domain/model"
)

// --- моки ---

type mockLocationRepo struct {
	findLTE   func(context.Context, *model.LTE) (*model.Location, error)
	findGSM   func(context.Context, *model.GSM) (*model.Location, error)
	findWCDMA func(context.Context, *model.WCDMA) (*model.Location, error)
	findWifi  func(context.Context, *model.Wifi) (*model.Location, error)
	findIP    func(context.Context, *model.Ip) (*model.Location, error)
}

func (m *mockLocationRepo) FindLTE(ctx context.Context, v *model.LTE) (*model.Location, error) {
	if m.findLTE != nil {
		return m.findLTE(ctx, v)
	}
	return nil, nil
}
func (m *mockLocationRepo) FindGSM(ctx context.Context, v *model.GSM) (*model.Location, error) {
	if m.findGSM != nil {
		return m.findGSM(ctx, v)
	}
	return nil, nil
}
func (m *mockLocationRepo) FindWCDMA(ctx context.Context, v *model.WCDMA) (*model.Location, error) {
	if m.findWCDMA != nil {
		return m.findWCDMA(ctx, v)
	}
	return nil, nil
}
func (m *mockLocationRepo) FindWifi(ctx context.Context, v *model.Wifi) (*model.Location, error) {
	if m.findWifi != nil {
		return m.findWifi(ctx, v)
	}
	return nil, nil
}
func (m *mockLocationRepo) FindIP(ctx context.Context, v *model.Ip) (*model.Location, error) {
	if m.findIP != nil {
		return m.findIP(ctx, v)
	}
	return nil, nil
}

type mockExternalRepo struct {
	findLTE   func(context.Context, *model.LTE) (*model.Location, error)
	findGSM   func(context.Context, *model.GSM) (*model.Location, error)
	findWCDMA func(context.Context, *model.WCDMA) (*model.Location, error)
}

func (m *mockExternalRepo) FindLTE(ctx context.Context, v *model.LTE) (*model.Location, error) {
	if m.findLTE != nil {
		return m.findLTE(ctx, v)
	}
	return nil, nil
}
func (m *mockExternalRepo) FindGSM(ctx context.Context, v *model.GSM) (*model.Location, error) {
	if m.findGSM != nil {
		return m.findGSM(ctx, v)
	}
	return nil, nil
}
func (m *mockExternalRepo) FindWCDMA(ctx context.Context, v *model.WCDMA) (*model.Location, error) {
	if m.findWCDMA != nil {
		return m.findWCDMA(ctx, v)
	}
	return nil, nil
}
func (m *mockExternalRepo) FetchSampleCells(ctx context.Context, limit int) ([]model.LocateRequest, error) {
	return nil, nil
}

type mockUpdateRepo struct {
	insertCell func(context.Context, *model.Cell, *model.Location, string, string, any) error
}

func (m *mockUpdateRepo) InsertCell(ctx context.Context, c *model.Cell, l *model.Location, source, objType string, raw any) error {
	if m.insertCell != nil {
		return m.insertCell(ctx, c, l, source, objType, raw)
	}
	return nil
}
func (m *mockUpdateRepo) InsertWifi(_ context.Context, _ *model.Wifi, _ *model.Location, _, _ string, _ any) error {
	return nil
}
func (m *mockUpdateRepo) InsertIP(_ context.Context, _ *model.Ip, _ *model.Location, _, _ string, _ any) error {
	return nil
}

type mockSessionRepo struct{}

func (m *mockSessionRepo) CreateIfNotExists(_ context.Context, _ string) error { return nil }
func (m *mockSessionRepo) SavePoint(_ context.Context, _ string, _, _ float64, _ float64, _ string, _, _ any) error {
	return nil
}
func (m *mockSessionRepo) ListSessions(_ context.Context, _, _ int) ([]model.SessionListRow, int64, error) {
	return nil, 0, nil
}
func (m *mockSessionRepo) GetSessionPoints(_ context.Context, _ string) ([]model.SessionPoint, error) {
	return nil, nil
}
func (m *mockSessionRepo) UpdatePoint(_ context.Context, _ int64, _, _ float64, _ float64) error {
	return nil
}

// syncGroup выполняет Go() синхронно — для тестов.
type syncGroup struct{}

func (s *syncGroup) Go(fn func()) { fn() }
func (s *syncGroup) Wait()        {}

// ---

func newLoc(lat, lon float64) *model.Location {
	return &model.Location{Point: model.Point{Lat: lat, Lon: lon}}
}

func newSvc(locate *mockLocationRepo, external *mockExternalRepo, update *mockUpdateRepo) *LocateService {
	return NewLocateService(locate, external, update, &mockSessionRepo{}, &syncGroup{})
}

func TestSearchBySource_CellOnly(t *testing.T) {
	svc := newSvc(
		&mockLocationRepo{
			findLTE: func(_ context.Context, _ *model.LTE) (*model.Location, error) {
				return newLoc(55.75, 37.61), nil
			},
		},
		&mockExternalRepo{},
		&mockUpdateRepo{},
	)
	req := &model.LocateRequest{Cell: []model.Cell{{LTE: &model.LTE{MCC: 1}}}}
	loc, err := svc.searchBySource(context.Background(), req)
	if err != nil || loc == nil {
		t.Fatalf("expected location, got err=%v loc=%v", err, loc)
	}
	if loc.Accuracy != float64(defaultCellAccuracy) {
		t.Errorf("expected accuracy=%f, got %f", float64(defaultCellAccuracy), loc.Accuracy)
	}
}

func TestSearchBySource_MetroIgnoresWifi(t *testing.T) {
	svc := newSvc(
		&mockLocationRepo{
			findLTE: func(_ context.Context, _ *model.LTE) (*model.Location, error) {
				loc := newLoc(55.75, 37.61)
				loc.FromMetro = true
				return loc, nil
			},
			findWifi: func(_ context.Context, _ *model.Wifi) (*model.Location, error) {
				t.Error("WiFi should not be called for metro cell")
				return newLoc(55.76, 37.62), nil
			},
		},
		&mockExternalRepo{},
		&mockUpdateRepo{},
	)
	req := &model.LocateRequest{
		Cell: []model.Cell{{LTE: &model.LTE{MCC: 1}}},
		Wifi: []model.Wifi{{BSSID: "aa:bb:cc:dd:ee:ff"}},
	}
	loc, err := svc.searchBySource(context.Background(), req)
	if err != nil || loc == nil {
		t.Fatalf("expected location, got err=%v loc=%v", err, loc)
	}
	if loc.Point.Lat != 55.75 {
		t.Errorf("expected cell lat=55.75, got %v", loc.Point.Lat)
	}
	if loc.Accuracy != float64(defaultCellAccuracy) {
		t.Errorf("expected accuracy=%f, got %f", float64(defaultCellAccuracy), loc.Accuracy)
	}
}

func TestSearchBySource_WifiConfirmsCell(t *testing.T) {
	// Cell Москва, WiFi рядом (~1км)
	svc := newSvc(
		&mockLocationRepo{
			findLTE: func(_ context.Context, _ *model.LTE) (*model.Location, error) {
				return newLoc(55.7558, 37.6173), nil
			},
			findWifi: func(_ context.Context, _ *model.Wifi) (*model.Location, error) {
				return newLoc(55.7650, 37.6173), nil
			},
		},
		&mockExternalRepo{},
		&mockUpdateRepo{},
	)
	req := &model.LocateRequest{
		Cell: []model.Cell{{LTE: &model.LTE{MCC: 1}}},
		Wifi: []model.Wifi{{BSSID: "aa:bb:cc:dd:ee:ff"}},
	}
	loc, err := svc.searchBySource(context.Background(), req)
	if err != nil || loc == nil {
		t.Fatalf("expected location, got err=%v loc=%v", err, loc)
	}
	if loc.Point.Lat != 55.7650 {
		t.Errorf("expected wifi lat, got %v", loc.Point.Lat)
	}
	if loc.Accuracy >= float64(defaultCellAccuracy) {
		t.Errorf("chord accuracy should be less than cell accuracy, got %f", loc.Accuracy)
	}
}

func TestSearchBySource_WifiFarFromCell(t *testing.T) {
	// WiFi в Питере (~600км от Москвы) — не подтверждает, берём Cell
	svc := newSvc(
		&mockLocationRepo{
			findLTE: func(_ context.Context, _ *model.LTE) (*model.Location, error) {
				return newLoc(55.7558, 37.6173), nil
			},
			findWifi: func(_ context.Context, _ *model.Wifi) (*model.Location, error) {
				return newLoc(59.9343, 30.3351), nil
			},
		},
		&mockExternalRepo{},
		&mockUpdateRepo{},
	)
	req := &model.LocateRequest{
		Cell: []model.Cell{{LTE: &model.LTE{MCC: 1}}},
		Wifi: []model.Wifi{{BSSID: "aa:bb:cc:dd:ee:ff"}},
	}
	loc, err := svc.searchBySource(context.Background(), req)
	if err != nil || loc == nil {
		t.Fatalf("expected location: err=%v loc=%v", err, loc)
	}
	if loc.Point.Lat != 55.7558 {
		t.Errorf("expected cell lat, got %v", loc.Point.Lat)
	}
	if loc.Accuracy != float64(defaultCellAccuracy) {
		t.Errorf("expected cell accuracy, got %f", loc.Accuracy)
	}
}

func TestSearchBySource_FallbackToIP(t *testing.T) {
	svc := newSvc(
		&mockLocationRepo{
			findIP: func(_ context.Context, _ *model.Ip) (*model.Location, error) {
				return newLoc(55.09, 32.84), nil
			},
		},
		&mockExternalRepo{},
		&mockUpdateRepo{},
	)
	req := &model.LocateRequest{IP: []model.Ip{{Address: "1.2.3.4"}}}
	loc, err := svc.searchBySource(context.Background(), req)
	if err != nil || loc == nil {
		t.Fatalf("expected location: err=%v loc=%v", err, loc)
	}
	if loc.Accuracy != float64(defaultIPAccuracy) {
		t.Errorf("expected IP accuracy, got %f", loc.Accuracy)
	}
}

func TestSearchBySource_ExternalDBSavesToUpdate(t *testing.T) {
	saved := false
	svc := newSvc(
		&mockLocationRepo{},
		&mockExternalRepo{
			findLTE: func(_ context.Context, _ *model.LTE) (*model.Location, error) {
				return newLoc(55.0, 37.0), nil
			},
		},
		&mockUpdateRepo{
			insertCell: func(_ context.Context, _ *model.Cell, _ *model.Location, source, _ string, _ any) error {
				if source != "external" {
					t.Errorf("expected source=external, got %s", source)
				}
				saved = true
				return nil
			},
		},
	)
	req := &model.LocateRequest{Cell: []model.Cell{{LTE: &model.LTE{MCC: 1}}}}
	if _, err := svc.searchBySource(context.Background(), req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !saved {
		t.Error("expected cell to be saved to updateRepo")
	}
}

func TestFindCellInRepo_UnknownType(t *testing.T) {
	_, err := findCellInRepo(context.Background(), &mockLocationRepo{}, model.Cell{})
	if err == nil {
		t.Error("expected error for unknown radio type")
	}
}
