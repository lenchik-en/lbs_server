package app

import (
	"testing"

	"github.com/lenchik-en/lbs_server/internal/api"
)

func validLocation() *api.Location {
	return &api.Location{
		Point:    api.Point{Lat: 55.75, Lon: 37.61},
		Accuracy: 50,
	}
}

func validLTECell() *api.Cell {
	return &api.Cell{
		LTE: &api.LTE{
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
		req     api.UpdateRequest
		wantErr bool
	}{
		{
			name: "valid cell update",
			req: api.UpdateRequest{
				Cell:     validLTECell(),
				Location: validLocation(),
			},
			wantErr: false,
		},
		{
			name: "no source provided",
			req: api.UpdateRequest{
				Location: validLocation(),
			},
			wantErr: true,
		},
		{
			name: "multiple sources",
			req: api.UpdateRequest{
				Cell:     validLTECell(),
				Wifi:     &api.Wifi{},
				Location: validLocation(),
			},
			wantErr: true,
		},
		{
			name: "empty cell source",
			req: api.UpdateRequest{
				Cell:     &api.Cell{},
				Location: validLocation(),
			},
			wantErr: true,
		},
		{
			name: "no location",
			req: api.UpdateRequest{
				Cell: validLTECell(),
			},
			wantErr: true,
		},
		{
			name: "invalid location",
			req: api.UpdateRequest{
				Cell: validLTECell(),
				Location: &api.Location{
					Point:    api.Point{Lat: 200, Lon: 0},
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
		loc     *api.Location
		wantErr bool
	}{
		{
			name: "valid location",
			loc: &api.Location{
				Point:    api.Point{Lat: 55.75, Lon: 37.61},
				Accuracy: 50,
			},
			wantErr: false,
		},
		{
			name: "zero coordinates allowed",
			loc: &api.Location{
				Point:    api.Point{Lat: 0, Lon: 0},
				Accuracy: 100,
			},
			wantErr: false,
		},
		{
			name: "latitude out of range",
			loc: &api.Location{
				Point: api.Point{
					Lat: 91,
					Lon: 0,
				},
				Accuracy: 10,
			},
			wantErr: true,
		},
		{
			name: "longitude out of range",
			loc: &api.Location{
				Point:    api.Point{Lat: 0, Lon: -181},
				Accuracy: 10,
			},
			wantErr: true,
		},
		{
			name: "accuracy must be positive",
			loc: &api.Location{
				Point:    api.Point{Lat: 10, Lon: 10},
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
