package app

import (
	"testing"

	"github.com/lenchik-en/lbs_server/internal/models"
)

func validLocation() *models.Location {
	return &models.Location{
		Point:    models.Point{Lat: 55.75, Lon: 37.61},
		Accuracy: 50,
	}
}

func validLTECell() *models.Cell {
	return &models.Cell{
		LTE: &models.LTE{
			MCC: 250,
			MNC: 99,
			TAC: 123,
			CI:  456,
		},
	}
}

func TestValidateUpdate(t *testing.T) {
	a := &App{}

	tests := []struct {
		name    string
		req     models.UpdateRequest
		wantErr bool
	}{
		{
			name: "valid cell update",
			req: models.UpdateRequest{
				Cell:     validLTECell(),
				Location: validLocation(),
			},
			wantErr: false,
		},
		{
			name: "no source provided",
			req: models.UpdateRequest{
				Location: validLocation(),
			},
			wantErr: true,
		},
		{
			name: "multiple sources",
			req: models.UpdateRequest{
				Cell:     validLTECell(),
				Wifi:     &models.Wifi{},
				Location: validLocation(),
			},
			wantErr: true,
		},
		{
			name: "empty cell source",
			req: models.UpdateRequest{
				Cell:     &models.Cell{},
				Location: validLocation(),
			},
			wantErr: true,
		},
		{
			name: "no location",
			req: models.UpdateRequest{
				Cell: validLTECell(),
			},
			wantErr: true,
		},
		{
			name: "invalid location",
			req: models.UpdateRequest{
				Cell: validLTECell(),
				Location: &models.Location{
					Point:    models.Point{Lat: 200, Lon: 0},
					Accuracy: 10,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := a.validateUpdate(tt.req)

			if tt.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateLocation(t *testing.T) {
	a, tests := &App{}, []struct {
		name    string
		loc     *models.Location
		wantErr bool
	}{
		{
			name: "valid location",
			loc: &models.Location{
				Point:    models.Point{Lat: 55.75, Lon: 37.61},
				Accuracy: 50,
			},
			wantErr: false,
		},
		{
			name: "zero coordinates allowed",
			loc: &models.Location{
				Point:    models.Point{Lat: 0, Lon: 0},
				Accuracy: 100,
			},
			wantErr: false,
		},
		{
			name: "latitude out of range",
			loc: &models.Location{
				Point: models.Point{
					Lat: 91,
					Lon: 0,
				},
				Accuracy: 10,
			},
			wantErr: true,
		},
		{
			name: "longitude out of range",
			loc: &models.Location{
				Point:    models.Point{Lat: 0, Lon: -181},
				Accuracy: 10,
			},
			wantErr: true,
		},
		{
			name: "accuracy must be positive",
			loc: &models.Location{
				Point:    models.Point{Lat: 10, Lon: 10},
				Accuracy: 0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := a.validateLocation(tt.loc)

			if tt.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
