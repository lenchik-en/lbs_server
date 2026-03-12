package app

import (
	"context"
	"database/sql"
	"testing"

	"github.com/lenchik-en/lbs_server/internal/models"
)

type mockCellFinder struct {
	findLTE       func(ctx context.Context, lte *models.LTE) (*models.Location, error)
	findGSM       func(ctx context.Context, gsm *models.GSM) (*models.Location, error)
	findWCDMA     func(ctx context.Context, wcdma *models.WCDMA) (*models.Location, error)
	getConnection func() *sql.DB
}

type mockUpdateDB struct {
	insertCell func(ctx context.Context, cell *models.Cell, loc *models.Location, source string, rawJSON any) error
}

func (m *mockUpdateDB) InsertCell(ctx context.Context, cell *models.Cell, loc *models.Location, source string, rawJSON any) error {
	if m.insertCell != nil {
		return m.insertCell(ctx, cell, loc, source, rawJSON)
	}
	return nil
}

func (m *mockUpdateDB) GetConnection() *sql.DB {
	return nil
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

func (m *mockCellFinder) GetConnection() *sql.DB {
	return nil
}

func TestFindLocation(t *testing.T) {
	ctx := context.Background()

	location := &models.Location{
		Point:    models.Point{Lat: 10, Lon: 20},
		Accuracy: 50,
	}

	tests := []struct {
		name       string
		locateDB   *mockCellFinder
		externalDB *mockCellFinder
		updateDB   *mockUpdateDB
		cell       models.Cell
		wantFound  bool
		wantErr    bool
	}{
		{
			name: "found in locateDB",
			locateDB: &mockCellFinder{
				findLTE: func(ctx context.Context, lte *models.LTE) (*models.Location, error) {
					return location, nil
				},
			},
			externalDB: &mockCellFinder{},
			updateDB:   &mockUpdateDB{},
			cell: models.Cell{
				LTE: &models.LTE{MCC: 1},
			},
			wantFound: true,
		},
		{
			name: "found in externalDB",
			locateDB: &mockCellFinder{
				findLTE: func(ctx context.Context, lte *models.LTE) (*models.Location, error) {
					return nil, nil
				},
			},
			externalDB: &mockCellFinder{
				findLTE: func(ctx context.Context, lte *models.LTE) (*models.Location, error) {
					return location, nil
				},
			},
			updateDB: &mockUpdateDB{},
			cell: models.Cell{
				LTE: &models.LTE{MCC: 1},
			},
			wantFound: true,
		},
		{
			name: "found in externalDB triggers save with source=external",
			locateDB: &mockCellFinder{
				findLTE: func(ctx context.Context, lte *models.LTE) (*models.Location, error) {
					return nil, nil
				},
			},
			externalDB: &mockCellFinder{
				findLTE: func(ctx context.Context, lte *models.LTE) (*models.Location, error) {
					return location, nil
				},
			},
			updateDB: &mockUpdateDB{
				insertCell: func(ctx context.Context, cell *models.Cell, loc *models.Location, source string, rawJSON any) error {
					if source != "external" {
						t.Errorf("expected source=external, got %s", source)
					}
					return nil
				},
			},
			cell: models.Cell{
				LTE: &models.LTE{MCC: 1},
			},
			wantFound: true,
		},
		{
			name:       "not found anywhere",
			locateDB:   &mockCellFinder{},
			externalDB: &mockCellFinder{},
			updateDB:   &mockUpdateDB{},
			cell: models.Cell{
				LTE: &models.LTE{MCC: 1},
			},
			wantFound: false,
		},
		{
			name: "locateDB error",
			locateDB: &mockCellFinder{
				findLTE: func(ctx context.Context, lte *models.LTE) (*models.Location, error) {
					return nil, context.Canceled
				},
			},
			externalDB: &mockCellFinder{},
			updateDB:   &mockUpdateDB{},
			cell: models.Cell{
				LTE: &models.LTE{MCC: 1},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &App{
				locateDB:   tt.locateDB,
				externalDB: tt.externalDB,
				updateDB:   tt.updateDB,
			}

			loc, err := a.findLocation(ctx, tt.cell)

			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantFound && loc == nil {
				t.Fatal("expected location, got nil")
			}
			if !tt.wantFound && loc != nil {
				t.Fatal("expected nil location")
			}
		})
	}
}

func TestFindInDB(t *testing.T) {
	ctx := context.Background()
	a := &App{}

	location := &models.Location{
		Point:    models.Point{Lat: 1, Lon: 2},
		Accuracy: 100,
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
			cell: models.Cell{
				LTE: &models.LTE{MCC: 1},
			},
			mock: &mockCellFinder{
				findLTE: func(ctx context.Context, lte *models.LTE) (*models.Location, error) {
					return location, nil
				},
			},
			wantLoc: true,
		},
		{
			name: "GSM found",
			cell: models.Cell{
				GSM: &models.GSM{MCC: 1},
			},
			mock: &mockCellFinder{
				findGSM: func(ctx context.Context, gsm *models.GSM) (*models.Location, error) {
					return location, nil
				},
			},
			wantLoc: true,
		},
		{
			name:    "unknown radio",
			cell:    models.Cell{},
			mock:    &mockCellFinder{},
			wantErr: true,
		},
		{
			name: "finder returns error",
			cell: models.Cell{
				LTE: &models.LTE{MCC: 1},
			},
			mock: &mockCellFinder{
				findLTE: func(ctx context.Context, lte *models.LTE) (*models.Location, error) {
					return nil, context.DeadlineExceeded
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loc, err := a.findInDB(ctx, tt.mock, tt.cell)

			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantLoc && loc == nil {
				t.Fatal("expected location, got nil")
			}
			if !tt.wantLoc && loc != nil {
				t.Fatal("expected nil location")
			}
		})
	}
}
