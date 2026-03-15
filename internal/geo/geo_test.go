package geo

import (
	"math"
	"testing"
)

func TestHaversineDistance(t *testing.T) {
	tests := []struct {
		name                   string
		lat1, lon1, lat2, lon2 float64
		wantMin, wantMax       float64 // допустимый диапазон в метрах
	}{
		{
			name: "same point",
			lat1: 55.7558, lon1: 37.6173,
			lat2: 55.7558, lon2: 37.6173,
			wantMin: 0, wantMax: 0,
		},
		{
			name: "Moscow to Saint Petersburg (~634 km)",
			lat1: 55.7558, lon1: 37.6173,
			lat2: 59.9343, lon2: 30.3351,
			wantMin: 630_000, wantMax: 640_000,
		},
		{
			name: "across 180th meridian (~15 km)",
			lat1: 50.0, lon1: 179.9,
			lat2: 50.0, lon2: -179.9,
			wantMin: 14_000, wantMax: 15_000,
		},
		{
			name: "southern hemisphere — Santiago to Buenos Aires (~1100 km)",
			lat1: -33.45, lon1: -70.67,
			lat2: -34.60, lon2: -58.38,
			wantMin: 1_100_000, wantMax: 1_150_000,
		},
		{
			name: "negative to positive lat",
			lat1: -1.0, lon1: 36.8,
			lat2: 1.0, lon2: 36.8,
			wantMin: 220_000, wantMax: 224_000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HaversineDistance(tt.lat1, tt.lon1, tt.lat2, tt.lon2)
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("HaversineDistance() = %f, want [%f, %f]", got, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestChordAccuracy(t *testing.T) {
	const minAcc = 1.0

	tests := []struct {
		name    string
		d       float64
		r1      float64
		r2      float64
		wantMin float64
		wantMax float64
	}{
		{
			name: "d=0 concentric circles",
			d:    0, r1: 1000, r2: 100,
			wantMin: 100, wantMax: 100,
		},
		{
			name: "d=0 equal radii",
			d:    0, r1: 100, r2: 100,
			wantMin: 100, wantMax: 100,
		},
		{
			name: "d microscopic (one inside other)",
			d:    0.001, r1: 1000, r2: 100,
			wantMin: 100, wantMax: 100,
		},
		{
			name: "d < |r1-r2| (well inside)",
			d:    500, r1: 1000, r2: 100,
			wantMin: 100, wantMax: 100,
		},
		{
			name: "normal intersection d=950",
			d:    950, r1: 1000, r2: 100,
			// x = (950^2 + 1000^2 - 100^2) / (2*950) = (902500+1000000-10000)/1900 ≈ 996.05
			// h = sqrt(1000000 - 996.05^2) = sqrt(7885) ≈ 88.8
			wantMin: 88, wantMax: 90,
		},
		{
			name: "tangent externally d=r1+r2",
			d:    1100, r1: 1000, r2: 100,
			wantMin: 1, wantMax: 1, // floor
		},
		{
			name: "no intersection d > r1+r2",
			d:    2000, r1: 1000, r2: 100,
			wantMin: 100, wantMax: 100,
		},
		{
			name: "reversed radii order (r1 < r2)",
			d:    500, r1: 100, r2: 1000,
			wantMin: 100, wantMax: 100,
		},
		{
			name: "equal radii normal intersection",
			d:    100, r1: 1000, r2: 1000,
			// x = (10000 + 1000000 - 1000000) / 200 = 50
			// h = sqrt(1000000 - 2500) = sqrt(997500) ≈ 998.75
			wantMin: 998, wantMax: 1000,
		},
		{
			name: "barely intersecting (d just under r1+r2)",
			d:    1099, r1: 1000, r2: 100,
			// should give a small positive h, > minAccuracy
			wantMin: 1, wantMax: 50,
		},
		{
			name: "float precision edge: d ≈ |r1-r2|",
			d:    900.0000001, r1: 1000, r2: 100,
			wantMin: 1, wantMax: 100, // should not NaN/panic
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ChordAccuracy(tt.d, tt.r1, tt.r2, minAcc)

			if math.IsNaN(got) || math.IsInf(got, 0) {
				t.Fatalf("ChordAccuracy() = %f (NaN or Inf)", got)
			}
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("ChordAccuracy() = %f, want [%f, %f]", got, tt.wantMin, tt.wantMax)
			}
		})
	}
}
