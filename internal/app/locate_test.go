package app

import (
	"context"
	"testing"

	"github.com/lenchik-en/lbs_server/internal/api"
)

type mockCellFinder struct {
	findLTE   func(ctx context.Context, lte *api.LTE) (*api.Location, error)
	findGSM   func(ctx context.Context, gsm *api.GSM) (*api.Location, error)
	findWCDMA func(ctx context.Context, wcdma *api.WCDMA) (*api.Location, error)
}

func (m *mockCellFinder) FindLTE(ctx context.Context, lte *api.LTE) (*api.Location, error) {
	if m.findLTE != nil {
		return m.findLTE(ctx, lte)
	}
	return nil, nil
}

func (m *mockCellFinder) FindGSM(ctx context.Context, gsm *api.GSM) (*api.Location, error) {
	if m.findGSM != nil {
		return m.findGSM(ctx, gsm)
	}
	return nil, nil
}

func (m *mockCellFinder) FindWCDMA(ctx context.Context, wcdma *api.WCDMA) (*api.Location, error) {
	if m.findWCDMA != nil {
		return m.findWCDMA(ctx, wcdma)
	}
	return nil, nil
}

func TestFindLocation(t *testing.T) {
	ctx := context.Background()

	location := &api.Location{
		Point:    api.Point{Lat: 10, Lon: 20},
		Accuracy: 50,
	}

	tests := []struct {
		name       string
		locateDB   *mockCellFinder
		externalDB *mockCellFinder
		cell       api.Cell
		wantFound  bool
		wantErr    bool
	}{
		{
			name: "found in locateDB",
			locateDB: &mockCellFinder{
				findLTE: func(ctx context.Context, lte *api.LTE) (*api.Location, error) {
					return location, nil
				},
			},
			externalDB: &mockCellFinder{},
			cell: api.Cell{
				LTE: &api.LTE{MCC: 1},
			},
			wantFound: true,
		},
		{
			name: "found in externalDB",
			locateDB: &mockCellFinder{
				findLTE: func(ctx context.Context, lte *api.LTE) (*api.Location, error) {
					return nil, nil
				},
			},
			externalDB: &mockCellFinder{
				findLTE: func(ctx context.Context, lte *api.LTE) (*api.Location, error) {
					return location, nil
				},
			},
			cell: api.Cell{
				LTE: &api.LTE{MCC: 1},
			},
			wantFound: true,
		},
		{
			name:       "not found anywhere",
			locateDB:   &mockCellFinder{},
			externalDB: &mockCellFinder{},
			cell: api.Cell{
				LTE: &api.LTE{MCC: 1},
			},
			wantFound: false,
		},
		{
			name: "locateDB error",
			locateDB: &mockCellFinder{
				findLTE: func(ctx context.Context, lte *api.LTE) (*api.Location, error) {
					return nil, context.Canceled
				},
			},
			externalDB: &mockCellFinder{},
			cell: api.Cell{
				LTE: &api.LTE{MCC: 1},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &App{
				locateDB:   tt.locateDB,
				externalDB: tt.externalDB,
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

	location := &api.Location{
		Point:    api.Point{Lat: 1, Lon: 2},
		Accuracy: 100,
	}

	tests := []struct {
		name    string
		cell    api.Cell
		mock    *mockCellFinder
		wantLoc bool
		wantErr bool
	}{
		{
			name: "LTE found",
			cell: api.Cell{
				LTE: &api.LTE{MCC: 1},
			},
			mock: &mockCellFinder{
				findLTE: func(ctx context.Context, lte *api.LTE) (*api.Location, error) {
					return location, nil
				},
			},
			wantLoc: true,
		},
		{
			name: "GSM found",
			cell: api.Cell{
				GSM: &api.GSM{MCC: 1},
			},
			mock: &mockCellFinder{
				findGSM: func(ctx context.Context, gsm *api.GSM) (*api.Location, error) {
					return location, nil
				},
			},
			wantLoc: true,
		},
		{
			name:    "unknown radio",
			cell:    api.Cell{},
			mock:    &mockCellFinder{},
			wantErr: true,
		},
		{
			name: "finder returns error",
			cell: api.Cell{
				LTE: &api.LTE{MCC: 1},
			},
			mock: &mockCellFinder{
				findLTE: func(ctx context.Context, lte *api.LTE) (*api.Location, error) {
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
