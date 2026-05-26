package service

import (
	"context"
	"errors"
	"testing"

	"github.com/lenchik-en/lbs_server/internal/domain/model"
)

// capturingUpdateRepo records all insert calls.
type capturingUpdateRepo struct {
	cellCount int
	wifiCount int
	ipCount   int
	cellErr   error
	wifiErr   error
	ipErr     error
}

func (r *capturingUpdateRepo) InsertCell(_ context.Context, _ *model.Cell, _ *model.Location, _, _ string, _ any) error {
	r.cellCount++
	return r.cellErr
}

func (r *capturingUpdateRepo) InsertWifi(_ context.Context, _ *model.Wifi, _ *model.Location, _, _ string, _ any) error {
	r.wifiCount++
	return r.wifiErr
}

func (r *capturingUpdateRepo) InsertIP(_ context.Context, _ *model.Ip, _ *model.Location, _, _ string, _ any) error {
	r.ipCount++
	return r.ipErr
}

// trackingSessionRepo records whether session methods were called.
type trackingSessionRepo struct {
	createCalled bool
	saveCalled   bool
	createErr    error
	saveErr      error
}

func (r *trackingSessionRepo) CreateIfNotExists(_ context.Context, _ string) error {
	r.createCalled = true
	return r.createErr
}

func (r *trackingSessionRepo) SavePoint(_ context.Context, _ string, _, _ float64, _ float64, _ string, _, _ any) error {
	r.saveCalled = true
	return r.saveErr
}

func (r *trackingSessionRepo) ListSessions(_ context.Context, _, _ int) ([]model.SessionListRow, int64, error) {
	return nil, 0, nil
}

func (r *trackingSessionRepo) GetSessionPoints(_ context.Context, _ string) ([]model.SessionPoint, error) {
	return nil, nil
}
func (r *trackingSessionRepo) UpdatePoint(_ context.Context, _ int64, _, _ float64, _ float64) error {
	return nil
}

func validUpdateReq() *model.UpdateRequest {
	return &model.UpdateRequest{
		Cell: []model.Cell{{LTE: &model.LTE{MCC: 250, MNC: 1}}},
		Location: &model.Location{
			Point:    model.Point{Lat: 55.75, Lon: 37.61},
			Accuracy: 100,
		},
	}
}

// --- Validate: location ---

func TestValidate_NilLocation(t *testing.T) {
	svc := NewUpdateService(&capturingUpdateRepo{}, &trackingSessionRepo{})
	err := svc.Validate(&model.UpdateRequest{Cell: []model.Cell{{LTE: &model.LTE{}}}})
	if err == nil {
		t.Fatal("expected error for nil location")
	}
}

func TestValidate_LatOutOfRange(t *testing.T) {
	for _, lat := range []float64{-90.001, -91, -180, 90.001, 91, 181} {
		req := &model.UpdateRequest{
			Location: &model.Location{Point: model.Point{Lat: lat, Lon: 0}, Accuracy: 1},
			Cell:     []model.Cell{{LTE: &model.LTE{}}},
		}
		if err := NewUpdateService(&capturingUpdateRepo{}, &trackingSessionRepo{}).Validate(req); err == nil {
			t.Errorf("expected error for lat=%v", lat)
		}
	}
}

func TestValidate_LatBoundaryValid(t *testing.T) {
	for _, lat := range []float64{-90, 0, 90} {
		req := &model.UpdateRequest{
			Location: &model.Location{Point: model.Point{Lat: lat, Lon: 0}, Accuracy: 1},
			Cell:     []model.Cell{{LTE: &model.LTE{}}},
		}
		if err := NewUpdateService(&capturingUpdateRepo{}, &trackingSessionRepo{}).Validate(req); err != nil {
			t.Errorf("unexpected error for lat=%v: %v", lat, err)
		}
	}
}

func TestValidate_LonOutOfRange(t *testing.T) {
	for _, lon := range []float64{-180.001, -181, 180.001, 181} {
		req := &model.UpdateRequest{
			Location: &model.Location{Point: model.Point{Lat: 0, Lon: lon}, Accuracy: 1},
			Cell:     []model.Cell{{LTE: &model.LTE{}}},
		}
		if err := NewUpdateService(&capturingUpdateRepo{}, &trackingSessionRepo{}).Validate(req); err == nil {
			t.Errorf("expected error for lon=%v", lon)
		}
	}
}

func TestValidate_LonBoundaryValid(t *testing.T) {
	for _, lon := range []float64{-180, 0, 180} {
		req := &model.UpdateRequest{
			Location: &model.Location{Point: model.Point{Lat: 0, Lon: lon}, Accuracy: 1},
			Cell:     []model.Cell{{LTE: &model.LTE{}}},
		}
		if err := NewUpdateService(&capturingUpdateRepo{}, &trackingSessionRepo{}).Validate(req); err != nil {
			t.Errorf("unexpected error for lon=%v: %v", lon, err)
		}
	}
}

func TestValidate_AccuracyInvalid(t *testing.T) {
	for _, acc := range []float64{0, -1, -100} {
		req := &model.UpdateRequest{
			Location: &model.Location{Point: model.Point{Lat: 0, Lon: 0}, Accuracy: acc},
			Cell:     []model.Cell{{LTE: &model.LTE{}}},
		}
		if err := NewUpdateService(&capturingUpdateRepo{}, &trackingSessionRepo{}).Validate(req); err == nil {
			t.Errorf("expected error for accuracy=%v", acc)
		}
	}
}

func TestValidate_AccuracyBoundaryValid(t *testing.T) {
	req := &model.UpdateRequest{
		Location: &model.Location{Point: model.Point{Lat: 0, Lon: 0}, Accuracy: 1},
		Cell:     []model.Cell{{LTE: &model.LTE{}}},
	}
	if err := NewUpdateService(&capturingUpdateRepo{}, &trackingSessionRepo{}).Validate(req); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// --- Validate: type ---

func TestValidate_InvalidType(t *testing.T) {
	req := validUpdateReq()
	req.Type = "unknown_type"
	if err := NewUpdateService(&capturingUpdateRepo{}, &trackingSessionRepo{}).Validate(req); err == nil {
		t.Error("expected error for invalid type")
	}
}

func TestValidate_ValidTypes(t *testing.T) {
	for _, typ := range []model.ObjectType{model.ObjectTypeMetro, model.ObjectTypeBuilding, model.ObjectTypeStreet, ""} {
		req := validUpdateReq()
		req.Type = typ
		if err := NewUpdateService(&capturingUpdateRepo{}, &trackingSessionRepo{}).Validate(req); err != nil {
			t.Errorf("unexpected error for type=%q: %v", typ, err)
		}
	}
}

// --- Validate: sources ---

func TestValidate_NoSources(t *testing.T) {
	req := &model.UpdateRequest{
		Location: &model.Location{Point: model.Point{Lat: 0, Lon: 0}, Accuracy: 1},
	}
	if err := NewUpdateService(&capturingUpdateRepo{}, &trackingSessionRepo{}).Validate(req); err == nil {
		t.Error("expected error for no sources")
	}
}

func TestValidate_WifiOnlyValid(t *testing.T) {
	req := &model.UpdateRequest{
		Location: &model.Location{Point: model.Point{Lat: 0, Lon: 0}, Accuracy: 1},
		Wifi:     []model.Wifi{{BSSID: "aa:bb:cc:dd:ee:ff"}},
	}
	if err := NewUpdateService(&capturingUpdateRepo{}, &trackingSessionRepo{}).Validate(req); err != nil {
		t.Errorf("unexpected error for wifi-only: %v", err)
	}
}

func TestValidate_IPOnlyValid(t *testing.T) {
	req := &model.UpdateRequest{
		Location: &model.Location{Point: model.Point{Lat: 0, Lon: 0}, Accuracy: 1},
		IP:       []model.Ip{{Address: "8.8.8.8"}},
	}
	if err := NewUpdateService(&capturingUpdateRepo{}, &trackingSessionRepo{}).Validate(req); err != nil {
		t.Errorf("unexpected error for ip-only: %v", err)
	}
}

// --- Validate: cells ---

func TestValidate_CellNoRadioType(t *testing.T) {
	req := &model.UpdateRequest{
		Location: &model.Location{Point: model.Point{Lat: 0, Lon: 0}, Accuracy: 1},
		Cell:     []model.Cell{{}},
	}
	if err := NewUpdateService(&capturingUpdateRepo{}, &trackingSessionRepo{}).Validate(req); err == nil {
		t.Error("expected error for cell without radio type")
	}
}

func TestValidate_SecondCellNoRadioType(t *testing.T) {
	req := &model.UpdateRequest{
		Location: &model.Location{Point: model.Point{Lat: 0, Lon: 0}, Accuracy: 1},
		Cell: []model.Cell{
			{LTE: &model.LTE{}},
			{}, // no radio type
		},
	}
	if err := NewUpdateService(&capturingUpdateRepo{}, &trackingSessionRepo{}).Validate(req); err == nil {
		t.Error("expected error for second cell without radio type")
	}
}

func TestValidate_AllRadioTypes(t *testing.T) {
	cases := []model.Cell{
		{LTE: &model.LTE{MCC: 1}},
		{GSM: &model.GSM{MCC: 1}},
		{WCDMA: &model.WCDMA{MCC: 1}},
	}
	for _, cell := range cases {
		req := &model.UpdateRequest{
			Location: &model.Location{Point: model.Point{Lat: 0, Lon: 0}, Accuracy: 1},
			Cell:     []model.Cell{cell},
		}
		if err := NewUpdateService(&capturingUpdateRepo{}, &trackingSessionRepo{}).Validate(req); err != nil {
			t.Errorf("unexpected error for cell=%+v: %v", cell, err)
		}
	}
}

// --- Validate: IP ---

func TestValidate_IPEmptyAddress(t *testing.T) {
	req := &model.UpdateRequest{
		Location: &model.Location{Point: model.Point{Lat: 0, Lon: 0}, Accuracy: 1},
		IP:       []model.Ip{{Address: ""}},
	}
	if err := NewUpdateService(&capturingUpdateRepo{}, &trackingSessionRepo{}).Validate(req); err == nil {
		t.Error("expected error for empty IP address")
	}
}

func TestValidate_IPInvalidAddress(t *testing.T) {
	for _, addr := range []string{"not-an-ip", "256.0.0.1", "localhost"} {
		req := &model.UpdateRequest{
			Location: &model.Location{Point: model.Point{Lat: 0, Lon: 0}, Accuracy: 1},
			IP:       []model.Ip{{Address: addr}},
		}
		if err := NewUpdateService(&capturingUpdateRepo{}, &trackingSessionRepo{}).Validate(req); err == nil {
			t.Errorf("expected error for invalid IP=%q", addr)
		}
	}
}

func TestValidate_IPValidAddresses(t *testing.T) {
	for _, addr := range []string{"1.2.3.4", "192.168.1.1", "::1", "2001:db8::1"} {
		req := &model.UpdateRequest{
			Location: &model.Location{Point: model.Point{Lat: 0, Lon: 0}, Accuracy: 1},
			IP:       []model.Ip{{Address: addr}},
		}
		if err := NewUpdateService(&capturingUpdateRepo{}, &trackingSessionRepo{}).Validate(req); err != nil {
			t.Errorf("unexpected error for valid IP=%q: %v", addr, err)
		}
	}
}

func TestValidate_FullValid(t *testing.T) {
	req := &model.UpdateRequest{
		SessionUUID: "test-uuid",
		Type:        model.ObjectTypeBuilding,
		Cell: []model.Cell{
			{LTE: &model.LTE{MCC: 250, MNC: 1}},
			{GSM: &model.GSM{MCC: 250, MNC: 2}},
		},
		Wifi:     []model.Wifi{{BSSID: "aa:bb:cc:dd:ee:ff"}},
		IP:       []model.Ip{{Address: "8.8.8.8"}},
		Location: &model.Location{Point: model.Point{Lat: 55.75, Lon: 37.61}, Accuracy: 50},
	}
	if err := NewUpdateService(&capturingUpdateRepo{}, &trackingSessionRepo{}).Validate(req); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// --- Process ---

func TestProcess_InsertsAllSources(t *testing.T) {
	repo := &capturingUpdateRepo{}
	svc := NewUpdateService(repo, &trackingSessionRepo{})
	req := &model.UpdateRequest{
		Cell: []model.Cell{{LTE: &model.LTE{}}, {GSM: &model.GSM{}}},
		Wifi: []model.Wifi{{BSSID: "aa"}, {BSSID: "bb"}},
		IP:   []model.Ip{{Address: "1.2.3.4"}},
		Location: &model.Location{
			Point:    model.Point{Lat: 55.0, Lon: 37.0},
			Accuracy: 100,
		},
	}
	if err := svc.Process(context.Background(), req, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.cellCount != 2 {
		t.Errorf("expected 2 cell inserts, got %d", repo.cellCount)
	}
	if repo.wifiCount != 2 {
		t.Errorf("expected 2 wifi inserts, got %d", repo.wifiCount)
	}
	if repo.ipCount != 1 {
		t.Errorf("expected 1 ip insert, got %d", repo.ipCount)
	}
}

func TestProcess_SessionCreatedWhenUUIDProvided(t *testing.T) {
	sess := &trackingSessionRepo{}
	svc := NewUpdateService(&capturingUpdateRepo{}, sess)
	req := validUpdateReq()
	req.SessionUUID = "some-uuid"
	if err := svc.Process(context.Background(), req, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !sess.createCalled {
		t.Error("expected CreateIfNotExists to be called")
	}
	if !sess.saveCalled {
		t.Error("expected SavePoint to be called")
	}
}

func TestProcess_NoSessionWhenNoUUID(t *testing.T) {
	sess := &trackingSessionRepo{}
	svc := NewUpdateService(&capturingUpdateRepo{}, sess)
	req := validUpdateReq()
	req.SessionUUID = ""
	if err := svc.Process(context.Background(), req, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sess.createCalled {
		t.Error("CreateIfNotExists should not be called without UUID")
	}
	if sess.saveCalled {
		t.Error("SavePoint should not be called without UUID")
	}
}

func TestProcess_CellInsertError(t *testing.T) {
	repo := &capturingUpdateRepo{cellErr: errors.New("db error")}
	svc := NewUpdateService(repo, &trackingSessionRepo{})
	if err := svc.Process(context.Background(), validUpdateReq(), nil); err == nil {
		t.Error("expected error from cell insert")
	}
}

func TestProcess_WifiInsertError(t *testing.T) {
	repo := &capturingUpdateRepo{wifiErr: errors.New("db error")}
	svc := NewUpdateService(repo, &trackingSessionRepo{})
	req := &model.UpdateRequest{
		Wifi:     []model.Wifi{{BSSID: "aa:bb:cc:dd:ee:ff"}},
		Location: &model.Location{Point: model.Point{Lat: 0, Lon: 0}, Accuracy: 1},
	}
	if err := svc.Process(context.Background(), req, nil); err == nil {
		t.Error("expected error from wifi insert")
	}
}

func TestProcess_IPInsertError(t *testing.T) {
	repo := &capturingUpdateRepo{ipErr: errors.New("db error")}
	svc := NewUpdateService(repo, &trackingSessionRepo{})
	req := &model.UpdateRequest{
		IP:       []model.Ip{{Address: "1.2.3.4"}},
		Location: &model.Location{Point: model.Point{Lat: 0, Lon: 0}, Accuracy: 1},
	}
	if err := svc.Process(context.Background(), req, nil); err == nil {
		t.Error("expected error from IP insert")
	}
}

func TestProcess_SessionErrorsDoNotFailProcess(t *testing.T) {
	sess := &trackingSessionRepo{
		createErr: errors.New("session db down"),
		saveErr:   errors.New("session db down"),
	}
	svc := NewUpdateService(&capturingUpdateRepo{}, sess)
	req := validUpdateReq()
	req.SessionUUID = "some-uuid"
	if err := svc.Process(context.Background(), req, nil); err != nil {
		t.Errorf("session errors must not fail Process: %v", err)
	}
}

func TestProcess_CellInsertStopsOnFirstError(t *testing.T) {
	repo := &capturingUpdateRepo{cellErr: errors.New("fail")}
	svc := NewUpdateService(repo, &trackingSessionRepo{})
	req := &model.UpdateRequest{
		Cell: []model.Cell{{LTE: &model.LTE{}}, {GSM: &model.GSM{}}, {WCDMA: &model.WCDMA{}}},
		Location: &model.Location{
			Point:    model.Point{Lat: 0, Lon: 0},
			Accuracy: 1,
		},
	}
	svc.Process(context.Background(), req, nil)
	if repo.cellCount != 1 {
		t.Errorf("expected exactly 1 cell attempt before error, got %d", repo.cellCount)
	}
}
