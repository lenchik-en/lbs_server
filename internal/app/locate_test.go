package app

import (
	"context"
	"database/sql"
	"testing"

	"github.com/lenchik-en/lbs_server/internal/models"
)

// mockCellFinder реализует db.CellFinder (для ExternalDB и findCellInDB).
type mockCellFinder struct {
	findLTE   func(ctx context.Context, lte *models.LTE) (*models.Location, error)
	findGSM   func(ctx context.Context, gsm *models.GSM) (*models.Location, error)
	findWCDMA func(ctx context.Context, wcdma *models.WCDMA) (*models.Location, error)
}

func (m *mockCellFinder) FindLTE(ctx context.Context, lte *models.LTE) (*models.Location, error) {
	if m.findLTE != nil {
		return m.findLTE(ctx, lte)
	}
	return nil, nil
}

func (m *mockCellFinder) FindGSM(ctx context.Context, gsm *models.GSM) (*models.Location, error) {
	if m.findGSM != nil {
		return m.findGSM(ctx, gsm)
	}
	return nil, nil
}

func (m *mockCellFinder) FindWCDMA(ctx context.Context, wcdma *models.WCDMA) (*models.Location, error) {
	if m.findWCDMA != nil {
		return m.findWCDMA(ctx, wcdma)
	}
	return nil, nil
}

func (m *mockCellFinder) GetConnection() *sql.DB { return nil }

// mockFinder реализует db.Finder (для LocateDB: cell + wifi + ip).
type mockFinder struct {
	mockCellFinder
	findWifi func(ctx context.Context, wifi *models.Wifi) (*models.Location, error)
	findIP   func(ctx context.Context, ip *models.Ip) (*models.Location, error)
}

func (m *mockFinder) FindWifi(ctx context.Context, wifi *models.Wifi) (*models.Location, error) {
	if m.findWifi != nil {
		return m.findWifi(ctx, wifi)
	}
	return nil, nil
}

func (m *mockFinder) FindIP(ctx context.Context, ip *models.Ip) (*models.Location, error) {
	if m.findIP != nil {
		return m.findIP(ctx, ip)
	}
	return nil, nil
}

// mockUpdateDB реализует db.Inserter.
type mockUpdateDB struct {
	insertCell func(ctx context.Context, cell *models.Cell, loc *models.Location, source string, rawJSON any) error
	insertWifi func(ctx context.Context, wifi *models.Wifi, source string, rawJSON any) error
	insertIP   func(ctx context.Context, ip *models.Ip, source string, rawJSON any) error
}

func (m *mockUpdateDB) InsertCell(ctx context.Context, cell *models.Cell, loc *models.Location, source string, rawJSON any) error {
	if m.insertCell != nil {
		return m.insertCell(ctx, cell, loc, source, rawJSON)
	}
	return nil
}

func (m *mockUpdateDB) InsertWifi(ctx context.Context, wifi *models.Wifi, source string, rawJSON any) error {
	if m.insertWifi != nil {
		return m.insertWifi(ctx, wifi, source, rawJSON)
	}
	return nil
}

func (m *mockUpdateDB) InsertIP(ctx context.Context, ip *models.Ip, source string, rawJSON any) error {
	if m.insertIP != nil {
		return m.insertIP(ctx, ip, source, rawJSON)
	}
	return nil
}

func (m *mockUpdateDB) GetConnection() *sql.DB { return nil }

// mockSession реализует db.SessionManager.
type mockSession struct {
	createSession func(ctx context.Context, sessionUUID string) error
	savePoint     func(ctx context.Context, sessionUUID string, lat, lon float64, accuracy int, source string, rawReq, rawResp interface{}) error
}

func (m *mockSession) CreateSessionIfNotExists(ctx context.Context, sessionUUID string) error {
	if m.createSession != nil {
		return m.createSession(ctx, sessionUUID)
	}
	return nil
}

func (m *mockSession) SavePoint(ctx context.Context, sessionUUID string, lat, lon float64, accuracy int, source string, rawReq, rawResp interface{}) error {
	if m.savePoint != nil {
		return m.savePoint(ctx, sessionUUID, lat, lon, accuracy, source, rawReq, rawResp)
	}
	return nil
}

func TestFindCellInDB(t *testing.T) {
	ctx := context.Background()

	location := &models.Location{
		Point: models.Point{Lat: 1, Lon: 2},
	}

	tests := []struct {
		name    string
		cell    models.Cell
		mock    *mockCellFinder
		wantLoc bool
		wantErr bool
	}{
		{
			name: "LTE found",
			cell: models.Cell{LTE: &models.LTE{MCC: 1}},
			mock: &mockCellFinder{
				findLTE: func(_ context.Context, _ *models.LTE) (*models.Location, error) {
					return location, nil
				},
			},
			wantLoc: true,
		},
		{
			name: "GSM found",
			cell: models.Cell{GSM: &models.GSM{MCC: 1}},
			mock: &mockCellFinder{
				findGSM: func(_ context.Context, _ *models.GSM) (*models.Location, error) {
					return location, nil
				},
			},
			wantLoc: true,
		},
		{
			name: "WCDMA found",
			cell: models.Cell{WCDMA: &models.WCDMA{MCC: 1}},
			mock: &mockCellFinder{
				findWCDMA: func(_ context.Context, _ *models.WCDMA) (*models.Location, error) {
					return location, nil
				},
			},
			wantLoc: true,
		},
		{
			name:    "unknown radio type",
			cell:    models.Cell{},
			mock:    &mockCellFinder{},
			wantErr: true,
		},
		{
			name: "finder returns error",
			cell: models.Cell{LTE: &models.LTE{MCC: 1}},
			mock: &mockCellFinder{
				findLTE: func(_ context.Context, _ *models.LTE) (*models.Location, error) {
					return nil, context.DeadlineExceeded
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loc, err := findCellInDB(ctx, tt.mock, tt.cell)

			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantLoc && loc == nil {
				t.Fatal("expected location, got nil")
			}
			if !tt.wantLoc && !tt.wantErr && loc != nil {
				t.Fatal("expected nil location")
			}
		})
	}
}

func TestFindCellLocation(t *testing.T) {
	ctx := context.Background()

	location := &models.Location{
		Point: models.Point{Lat: 10, Lon: 20},
	}

	tests := []struct {
		name       string
		locateDB   *mockFinder
		externalDB *mockCellFinder
		updateDB   *mockUpdateDB
		cell       models.Cell
		wantFound  bool
		wantErr    bool
	}{
		{
			name: "found in locateDB",
			locateDB: &mockFinder{
				mockCellFinder: mockCellFinder{
					findLTE: func(_ context.Context, _ *models.LTE) (*models.Location, error) {
						return location, nil
					},
				},
			},
			externalDB: &mockCellFinder{},
			updateDB:   &mockUpdateDB{},
			cell:       models.Cell{LTE: &models.LTE{MCC: 1}},
			wantFound:  true,
		},
		{
			name: "found in externalDB",
			locateDB: &mockFinder{
				mockCellFinder: mockCellFinder{
					findLTE: func(_ context.Context, _ *models.LTE) (*models.Location, error) {
						return nil, nil
					},
				},
			},
			externalDB: &mockCellFinder{
				findLTE: func(_ context.Context, _ *models.LTE) (*models.Location, error) {
					return location, nil
				},
			},
			updateDB:  &mockUpdateDB{},
			cell:      models.Cell{LTE: &models.LTE{MCC: 1}},
			wantFound: true,
		},
		{
			name: "externalDB hit triggers save with source=external",
			locateDB: &mockFinder{
				mockCellFinder: mockCellFinder{
					findLTE: func(_ context.Context, _ *models.LTE) (*models.Location, error) {
						return nil, nil
					},
				},
			},
			externalDB: &mockCellFinder{
				findLTE: func(_ context.Context, _ *models.LTE) (*models.Location, error) {
					return location, nil
				},
			},
			updateDB: &mockUpdateDB{
				insertCell: func(_ context.Context, _ *models.Cell, _ *models.Location, source string, _ any) error {
					if source != "external" {
						t.Errorf("expected source=external, got %s", source)
					}
					return nil
				},
			},
			cell:      models.Cell{LTE: &models.LTE{MCC: 1}},
			wantFound: true,
		},
		{
			name: "not found anywhere",
			locateDB: &mockFinder{
				mockCellFinder: mockCellFinder{},
			},
			externalDB: &mockCellFinder{},
			updateDB:   &mockUpdateDB{},
			cell:       models.Cell{LTE: &models.LTE{MCC: 1}},
			wantFound:  false,
		},
		{
			name: "locateDB error",
			locateDB: &mockFinder{
				mockCellFinder: mockCellFinder{
					findLTE: func(_ context.Context, _ *models.LTE) (*models.Location, error) {
						return nil, context.Canceled
					},
				},
			},
			externalDB: &mockCellFinder{},
			updateDB:   &mockUpdateDB{},
			cell:       models.Cell{LTE: &models.LTE{MCC: 1}},
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &App{
				locateDB:   tt.locateDB,
				externalDB: tt.externalDB,
				updateDB:   tt.updateDB,
			}

			loc, err := a.findCellLocation(ctx, tt.cell)

			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantFound && loc == nil {
				t.Fatal("expected location, got nil")
			}
			if !tt.wantFound && !tt.wantErr && loc != nil {
				t.Fatal("expected nil location")
			}
		})
	}
}

// newLoc создаёт свежий Location для каждого вызова мока.
// Это критично: searchBySource мутирует Accuracy на указателе,
// и переиспользование одного объекта между тестами ломает изоляцию.
func newLoc(lat, lon float64) *models.Location {
	return &models.Location{Point: models.Point{Lat: lat, Lon: lon}}
}

func TestSearchBySource(t *testing.T) {
	ctx := context.Background()

	// Координаты для тестов:
	// Cell в Москве: 55.7558, 37.6173
	// WiFi рядом (~1.1 км): 55.7650, 37.6173
	// WiFi далеко (Питер, ~600 км): 59.9343, 30.3351
	// IP: 55.09, 32.84

	tests := []struct {
		name         string
		locateDB     *mockFinder
		updateDB     *mockUpdateDB
		req          models.LocateRequest
		wantNil      bool
		wantLat      float64
		wantAccuracy int
	}{
		{
			name: "cell only — accuracy=1000",
			locateDB: &mockFinder{
				mockCellFinder: mockCellFinder{
					findLTE: func(_ context.Context, _ *models.LTE) (*models.Location, error) {
						return newLoc(55.7558, 37.6173), nil
					},
				},
			},
			updateDB: &mockUpdateDB{},
			req: models.LocateRequest{
				Cell: []models.Cell{{LTE: &models.LTE{MCC: 1}}},
			},
			wantLat:      55.7558,
			wantAccuracy: 1000,
		},
		{
			name: "wifi only — accuracy=100",
			locateDB: &mockFinder{
				mockCellFinder: mockCellFinder{},
				findWifi: func(_ context.Context, _ *models.Wifi) (*models.Location, error) {
					return newLoc(55.7650, 37.6173), nil
				},
			},
			updateDB: &mockUpdateDB{},
			req: models.LocateRequest{
				Wifi: []models.Wifi{{BSSID: "aa:bb:cc:dd:ee:ff"}},
			},
			wantLat:      55.7650,
			wantAccuracy: 100,
		},
		{
			name: "ip only — accuracy=100",
			locateDB: &mockFinder{
				mockCellFinder: mockCellFinder{},
				findIP: func(_ context.Context, _ *models.Ip) (*models.Location, error) {
					return newLoc(55.09, 32.84), nil
				},
			},
			updateDB: &mockUpdateDB{},
			req: models.LocateRequest{
				IP: []models.Ip{{Address: "1.2.3.4"}},
			},
			wantLat:      55.09,
			wantAccuracy: 100,
		},
		{
			name: "cell + wifi near — wifi confirms cell, accuracy from chord",
			locateDB: &mockFinder{
				mockCellFinder: mockCellFinder{
					findLTE: func(_ context.Context, _ *models.LTE) (*models.Location, error) {
						return newLoc(55.7558, 37.6173), nil
					},
				},
				findWifi: func(_ context.Context, _ *models.Wifi) (*models.Location, error) {
					return newLoc(55.7650, 37.6173), nil
				},
			},
			updateDB: &mockUpdateDB{},
			req: models.LocateRequest{
				Cell: []models.Cell{{LTE: &models.LTE{MCC: 1}}},
				Wifi: []models.Wifi{{BSSID: "aa:bb:cc:dd:ee:ff"}},
			},
			wantLat: 55.7650, // wifi coordinate used
			// d≈1023m, x=(1023²+1000²-100²)/(2*1023)≈995.37, h=sqrt(1000²-995.37²)=sqrt(9239)≈96.12 → int=96
			wantAccuracy: 96,
		},
		{
			name: "cell + wifi far — wifi rejected, cell only accuracy=1000",
			locateDB: &mockFinder{
				mockCellFinder: mockCellFinder{
					findLTE: func(_ context.Context, _ *models.LTE) (*models.Location, error) {
						return newLoc(55.7558, 37.6173), nil
					},
				},
				findWifi: func(_ context.Context, _ *models.Wifi) (*models.Location, error) {
					return newLoc(59.9343, 30.3351), nil
				},
			},
			updateDB: &mockUpdateDB{},
			req: models.LocateRequest{
				Cell: []models.Cell{{LTE: &models.LTE{MCC: 1}}},
				Wifi: []models.Wifi{{BSSID: "aa:bb:cc:dd:ee:ff"}},
			},
			wantLat:      55.7558, // cell coordinate used
			wantAccuracy: 1000,
		},
		{
			name: "cell + wifi same coords (d=0) — accuracy=min(r1,r2)=100",
			locateDB: &mockFinder{
				mockCellFinder: mockCellFinder{
					findLTE: func(_ context.Context, _ *models.LTE) (*models.Location, error) {
						return newLoc(55.7558, 37.6173), nil
					},
				},
				findWifi: func(_ context.Context, _ *models.Wifi) (*models.Location, error) {
					return newLoc(55.7558, 37.6173), nil
				},
			},
			updateDB: &mockUpdateDB{},
			req: models.LocateRequest{
				Cell: []models.Cell{{LTE: &models.LTE{MCC: 1}}},
				Wifi: []models.Wifi{{BSSID: "aa:bb:cc:dd:ee:ff"}},
			},
			wantLat:      55.7558,
			wantAccuracy: 100,
		},
		{
			name: "nothing found — nil",
			locateDB: &mockFinder{
				mockCellFinder: mockCellFinder{},
			},
			updateDB: &mockUpdateDB{},
			req: models.LocateRequest{
				Cell: []models.Cell{{LTE: &models.LTE{MCC: 1}}},
			},
			wantNil: true,
		},
		{
			name: "cell not found, wifi not found — fallback to ip",
			locateDB: &mockFinder{
				mockCellFinder: mockCellFinder{},
				findIP: func(_ context.Context, _ *models.Ip) (*models.Location, error) {
					return newLoc(55.09, 32.84), nil
				},
			},
			updateDB: &mockUpdateDB{},
			req: models.LocateRequest{
				Cell: []models.Cell{{LTE: &models.LTE{MCC: 1}}},
				Wifi: []models.Wifi{{BSSID: "aa:bb:cc:dd:ee:ff"}},
				IP:   []models.Ip{{Address: "1.2.3.4"}},
			},
			wantLat:      55.09,
			wantAccuracy: 100,
		},
		{
			name: "cell not found, wifi found, ip present — wifi wins over ip",
			locateDB: &mockFinder{
				mockCellFinder: mockCellFinder{},
				findWifi: func(_ context.Context, _ *models.Wifi) (*models.Location, error) {
					return newLoc(55.7650, 37.6173), nil
				},
				findIP: func(_ context.Context, _ *models.Ip) (*models.Location, error) {
					return newLoc(55.09, 32.84), nil
				},
			},
			updateDB: &mockUpdateDB{},
			req: models.LocateRequest{
				Cell: []models.Cell{{LTE: &models.LTE{MCC: 1}}},
				Wifi: []models.Wifi{{BSSID: "aa:bb:cc:dd:ee:ff"}},
				IP:   []models.Ip{{Address: "1.2.3.4"}},
			},
			wantLat:      55.7650,
			wantAccuracy: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &App{
				locateDB:   tt.locateDB,
				externalDB: &mockCellFinder{},
				updateDB:   tt.updateDB,
				session:    &mockSession{},
			}

			resp, err := a.findLocation(ctx, &tt.req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantNil {
				if resp != nil {
					t.Fatalf("expected nil response, got %+v", resp)
				}
				return
			}

			if resp == nil {
				t.Fatal("expected response, got nil")
			}
			if resp.Location.Point.Lat != tt.wantLat {
				t.Errorf("expected lat=%v, got %v", tt.wantLat, resp.Location.Point.Lat)
			}
			if resp.Location.Accuracy != tt.wantAccuracy {
				t.Errorf("expected accuracy=%d, got %d", tt.wantAccuracy, resp.Location.Accuracy)
			}
			if resp.SessionUUID == "" {
				t.Error("expected non-empty SessionUUID")
			}
		})
	}
}
