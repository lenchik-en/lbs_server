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
func validLTECell() models.Cell {
	return models.Cell{
		LTE: &models.LTE{
			MCC:           250,
			MNC:           99,
			TAC:           1000,
			CI:            5632016,
			PCI:           100,
			EARFCN:        1300,
			Age:           0,
			TimingAdvance: 0,
		},
	}
}

func validGSMCell() models.Cell {
	return models.Cell{
		GSM: &models.GSM{
			MCC:           250,
			MNC:           1,
			LAC:           3604,
			CID:           3171,
			BSIC:          10,
			ARFCN:         100,
			Age:           0,
			TimingAdvance: 0,
		},
	}
}

func validWCDMACell() models.Cell {
	return models.Cell{
		WCDMA: &models.WCDMA{
			MCC:    250,
			MNC:    1,
			LAC:    1000,
			CID:    12345,
			PSC:    100,
			UARFCN: 2100,
			Age:    0,
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
			name:    "valid single LTE cell",
			req:     models.UpdateRequest{Cell: []models.Cell{validLTECell()}, Location: validLocation()},
			wantErr: false,
		},
		{
			name:    "valid single GSM cell",
			req:     models.UpdateRequest{Cell: []models.Cell{validGSMCell()}, Location: validLocation()},
			wantErr: false,
		},
		{
			name:    "valid single WCDMA cell",
			req:     models.UpdateRequest{Cell: []models.Cell{validWCDMACell()}, Location: validLocation()},
			wantErr: false,
		},
		{
			name: "valid multiple cells",
			req: models.UpdateRequest{
				Cell:     []models.Cell{validLTECell(), validGSMCell()},
				Location: validLocation(),
			},
			wantErr: false,
		},
		{
			name: "valid with sessionUUID",
			req: models.UpdateRequest{
				SessionUUID: "test-uuid",
				Cell:        []models.Cell{validLTECell()},
				Location:    validLocation(),
			},
			wantErr: false,
		},
		{
			name:    "no source provided",
			req:     models.UpdateRequest{Location: validLocation()},
			wantErr: true,
		},
		{
			name:    "empty cell slice",
			req:     models.UpdateRequest{Cell: []models.Cell{}, Location: validLocation()},
			wantErr: true,
		},
		{
			name:    "cell with no radio type",
			req:     models.UpdateRequest{Cell: []models.Cell{{}}, Location: validLocation()},
			wantErr: true,
		},
		{
			name:    "second cell has no radio type",
			req:     models.UpdateRequest{Cell: []models.Cell{validLTECell(), {}}, Location: validLocation()},
			wantErr: true,
		},
		{
			name:    "no location",
			req:     models.UpdateRequest{Cell: []models.Cell{validLTECell()}},
			wantErr: true,
		},
		{
			name: "invalid location lat",
			req: models.UpdateRequest{
				Cell:     []models.Cell{validLTECell()},
				Location: &models.Location{Point: models.Point{Lat: 200, Lon: 0}, Accuracy: 10},
			},
			wantErr: true,
		},
		{
			name:    "valid wifi source",
			req:     models.UpdateRequest{Wifi: []models.Wifi{{BSSID: "00:11:22:33:44:55"}}, Location: validLocation()},
			wantErr: false,
		},
		{
			name:    "valid ip source",
			req:     models.UpdateRequest{IP: []models.Ip{{Address: "1.2.3.4"}}, Location: validLocation()},
			wantErr: false,
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

func TestValidateIP(t *testing.T) {
	tests := []struct {
		name    string
		ip      models.Ip
		wantErr bool
	}{
		{name: "valid ipv4", ip: models.Ip{Address: "192.168.1.1"}, wantErr: false},
		{name: "valid ipv6", ip: models.Ip{Address: "2001:db8::1"}, wantErr: false},
		{name: "empty address", ip: models.Ip{Address: ""}, wantErr: true},
		{name: "invalid address", ip: models.Ip{Address: "not-an-ip"}, wantErr: true},
		{name: "invalid octets", ip: models.Ip{Address: "999.999.999.999"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateIP(0, tt.ip)
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
	a := &App{}
	tests := []struct {
		name    string
		loc     *models.Location
		wantErr bool
	}{
		{
			name:    "valid location",
			loc:     &models.Location{Point: models.Point{Lat: 55.75, Lon: 37.61}, Accuracy: 50},
			wantErr: false,
		},
		{
			name:    "zero coordinates allowed",
			loc:     &models.Location{Point: models.Point{Lat: 0, Lon: 0}, Accuracy: 100},
			wantErr: false,
		},
		{
			name:    "latitude out of range",
			loc:     &models.Location{Point: models.Point{Lat: 91, Lon: 0}, Accuracy: 10},
			wantErr: true,
		},
		{
			name:    "longitude out of range",
			loc:     &models.Location{Point: models.Point{Lat: 0, Lon: -181}, Accuracy: 10},
			wantErr: true,
		},
		{
			name:    "accuracy must be positive",
			loc:     &models.Location{Point: models.Point{Lat: 10, Lon: 10}, Accuracy: 0},
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
