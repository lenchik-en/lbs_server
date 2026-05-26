package service

import (
	"context"
	"errors"
	"testing"

	"github.com/lenchik-en/lbs_server/internal/domain/model"
)

// --- findCellInRepo: GSM и WCDMA ветки (покрывают 60% → 100%) ---

func TestFindCellInRepo_GSMCell(t *testing.T) {
	repo := &mockLocationRepo{
		findGSM: func(_ context.Context, _ *model.GSM) (*model.Location, error) {
			return newLoc(55.0, 37.0), nil
		},
	}
	loc, err := findCellInRepo(context.Background(), repo, model.Cell{GSM: &model.GSM{MCC: 250}})
	if err != nil || loc == nil {
		t.Fatalf("expected location for GSM: err=%v loc=%v", err, loc)
	}
}

func TestFindCellInRepo_WCDMACell(t *testing.T) {
	repo := &mockLocationRepo{
		findWCDMA: func(_ context.Context, _ *model.WCDMA) (*model.Location, error) {
			return newLoc(55.0, 37.0), nil
		},
	}
	loc, err := findCellInRepo(context.Background(), repo, model.Cell{WCDMA: &model.WCDMA{MCC: 250}})
	if err != nil || loc == nil {
		t.Fatalf("expected location for WCDMA: err=%v loc=%v", err, loc)
	}
}

// --- findWifi / findIP: ветки ошибок ---

func TestSearchBySource_WifiError(t *testing.T) {
	svc := newSvc(
		&mockLocationRepo{
			findWifi: func(_ context.Context, _ *model.Wifi) (*model.Location, error) {
				return nil, errors.New("wifi db error")
			},
		},
		&mockExternalRepo{},
		&mockUpdateRepo{},
	)
	req := &model.LocateRequest{Wifi: []model.Wifi{{BSSID: "aa:bb:cc:dd:ee:ff"}}}
	_, err := svc.searchBySource(context.Background(), req)
	if err == nil {
		t.Error("expected error from wifi repo")
	}
}

func TestSearchBySource_IPError(t *testing.T) {
	svc := newSvc(
		&mockLocationRepo{
			findIP: func(_ context.Context, _ *model.Ip) (*model.Location, error) {
				return nil, errors.New("ip db error")
			},
		},
		&mockExternalRepo{},
		&mockUpdateRepo{},
	)
	req := &model.LocateRequest{IP: []model.Ip{{Address: "1.2.3.4"}}}
	_, err := svc.searchBySource(context.Background(), req)
	if err == nil {
		t.Error("expected error from IP repo")
	}
}

// --- WiFi без cell (путь wifiLoc != nil после cellLoc == nil) ---

func TestSearchBySource_WifiOnly(t *testing.T) {
	svc := newSvc(
		&mockLocationRepo{
			findWifi: func(_ context.Context, _ *model.Wifi) (*model.Location, error) {
				return newLoc(55.75, 37.61), nil
			},
		},
		&mockExternalRepo{},
		&mockUpdateRepo{},
	)
	req := &model.LocateRequest{Wifi: []model.Wifi{{BSSID: "aa:bb:cc:dd:ee:ff"}}}
	loc, err := svc.searchBySource(context.Background(), req)
	if err != nil || loc == nil {
		t.Fatalf("expected location for wifi-only: err=%v loc=%v", err, loc)
	}
	if loc.Accuracy != float64(defaultWifiAccuracy) {
		t.Errorf("expected wifi accuracy=%f, got %f", float64(defaultWifiAccuracy), loc.Accuracy)
	}
}

// --- externalRepo ошибка в findCellLocation ---

func TestFindCellLocation_ExternalRepoError(t *testing.T) {
	svc := newSvc(
		&mockLocationRepo{}, // LocateDB возвращает nil
		&mockExternalRepo{
			findLTE: func(_ context.Context, _ *model.LTE) (*model.Location, error) {
				return nil, errors.New("external db error")
			},
		},
		&mockUpdateRepo{},
	)
	req := &model.LocateRequest{Cell: []model.Cell{{LTE: &model.LTE{MCC: 1}}}}
	_, err := svc.searchBySource(context.Background(), req)
	if err == nil {
		t.Error("expected error from external repo")
	}
}

// --- InsertCell ошибка при сохранении из ExternalDB (WARN, не возвращается) ---

func TestFindCellLocation_UpdateInsertError_IsLogged(t *testing.T) {
	svc := newSvc(
		&mockLocationRepo{},
		&mockExternalRepo{
			findLTE: func(_ context.Context, _ *model.LTE) (*model.Location, error) {
				return newLoc(55.0, 37.0), nil
			},
		},
		&mockUpdateRepo{
			insertCell: func(_ context.Context, _ *model.Cell, _ *model.Location, _, _ string, _ any) error {
				return errors.New("insert error")
			},
		},
	)
	req := &model.LocateRequest{Cell: []model.Cell{{LTE: &model.LTE{MCC: 1}}}}
	loc, err := svc.searchBySource(context.Background(), req)
	if err != nil {
		t.Errorf("insert error must be logged, not returned: %v", err)
	}
	if loc == nil {
		t.Error("expected location despite insert error")
	}
}

// --- GSM / WCDMA в ExternalDB → сохраняются в UpdateDB ---

func TestSearchBySource_ExternalGSM_SavesToUpdate(t *testing.T) {
	saved := false
	svc := newSvc(
		&mockLocationRepo{},
		&mockExternalRepo{
			findGSM: func(_ context.Context, _ *model.GSM) (*model.Location, error) {
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
	req := &model.LocateRequest{Cell: []model.Cell{{GSM: &model.GSM{MCC: 250}}}}
	svc.searchBySource(context.Background(), req)
	if !saved {
		t.Error("expected GSM cell from ExternalDB to be saved to updateRepo")
	}
}

func TestSearchBySource_ExternalWCDMA_SavesToUpdate(t *testing.T) {
	saved := false
	svc := newSvc(
		&mockLocationRepo{},
		&mockExternalRepo{
			findWCDMA: func(_ context.Context, _ *model.WCDMA) (*model.Location, error) {
				return newLoc(55.0, 37.0), nil
			},
		},
		&mockUpdateRepo{
			insertCell: func(_ context.Context, _ *model.Cell, _ *model.Location, source, _ string, _ any) error {
				saved = true
				return nil
			},
		},
	)
	req := &model.LocateRequest{Cell: []model.Cell{{WCDMA: &model.WCDMA{MCC: 250}}}}
	svc.searchBySource(context.Background(), req)
	if !saved {
		t.Error("expected WCDMA cell from ExternalDB to be saved to updateRepo")
	}
}

// --- findCell: несколько ячеек, первая не найдена, вторая найдена ---

func TestSearchBySource_FirstCellNotFound_SecondCellFound(t *testing.T) {
	calls := 0
	svc := newSvc(
		&mockLocationRepo{
			findLTE: func(_ context.Context, lte *model.LTE) (*model.Location, error) {
				calls++
				if lte.MCC == 1 {
					return nil, nil // первая ячейка не найдена
				}
				return newLoc(55.0, 37.0), nil // вторая найдена
			},
		},
		&mockExternalRepo{},
		&mockUpdateRepo{},
	)
	req := &model.LocateRequest{
		Cell: []model.Cell{
			{LTE: &model.LTE{MCC: 1}},
			{LTE: &model.LTE{MCC: 2}},
		},
	}
	loc, err := svc.searchBySource(context.Background(), req)
	if err != nil || loc == nil {
		t.Fatalf("expected location from second cell: err=%v loc=%v", err, loc)
	}
	if calls != 2 {
		t.Errorf("expected 2 FindLTE calls, got %d", calls)
	}
}

// --- FindLocation: ошибки SessionRepo не должны останавливать обработку ---

func TestFindLocation_SessionCreateError_ReturnsLocation(t *testing.T) {
	svc := NewLocateService(
		&mockLocationRepo{
			findLTE: func(_ context.Context, _ *model.LTE) (*model.Location, error) {
				return newLoc(55.75, 37.61), nil
			},
		},
		&mockExternalRepo{},
		&mockUpdateRepo{},
		&trackingSessionRepo{createErr: errors.New("session error")},
		&syncGroup{},
	)
	req := &model.LocateRequest{Cell: []model.Cell{{LTE: &model.LTE{MCC: 1}}}}
	resp, err := svc.FindLocation(context.Background(), req)
	if err != nil || resp == nil {
		t.Errorf("session create error must not fail FindLocation: err=%v resp=%v", err, resp)
	}
}

func TestFindLocation_SessionSaveError_ReturnsLocation(t *testing.T) {
	svc := NewLocateService(
		&mockLocationRepo{
			findLTE: func(_ context.Context, _ *model.LTE) (*model.Location, error) {
				return newLoc(55.75, 37.61), nil
			},
		},
		&mockExternalRepo{},
		&mockUpdateRepo{},
		&trackingSessionRepo{saveErr: errors.New("save error")},
		&syncGroup{},
	)
	req := &model.LocateRequest{Cell: []model.Cell{{LTE: &model.LTE{MCC: 1}}}}
	resp, err := svc.FindLocation(context.Background(), req)
	if err != nil || resp == nil {
		t.Errorf("session save error must not fail FindLocation: err=%v resp=%v", err, resp)
	}
}

// --- Множественные WiFi: берётся первый найденный ---

func TestFindWifi_ReturnsFirstFound(t *testing.T) {
	callCount := 0
	svc := newSvc(
		&mockLocationRepo{
			findWifi: func(_ context.Context, w *model.Wifi) (*model.Location, error) {
				callCount++
				if w.BSSID == "first" {
					return newLoc(55.0, 37.0), nil
				}
				t.Errorf("second wifi should not be queried after first is found")
				return nil, nil
			},
		},
		&mockExternalRepo{},
		&mockUpdateRepo{},
	)
	req := &model.LocateRequest{
		Wifi: []model.Wifi{
			{BSSID: "first"},
			{BSSID: "second"},
		},
	}
	loc, err := svc.searchBySource(context.Background(), req)
	if err != nil || loc == nil {
		t.Fatalf("expected location: err=%v loc=%v", err, loc)
	}
	if callCount != 1 {
		t.Errorf("expected 1 wifi lookup, got %d", callCount)
	}
}
