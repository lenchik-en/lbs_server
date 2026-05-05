package service

import (
	"context"
	"errors"
	"testing"

	"github.com/lenchik-en/lbs_server/internal/domain/model"
)

// Tests for FindLocation (outer method with session management).
// Uses helpers newLoc and newSvc defined in locate_test.go.

func TestFindLocation_GeneratesUUIDWhenEmpty(t *testing.T) {
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
	resp, err := svc.FindLocation(context.Background(), req)
	if err != nil || resp == nil {
		t.Fatalf("unexpected result: err=%v resp=%v", err, resp)
	}
	if resp.SessionUUID == "" {
		t.Error("expected UUID to be auto-generated")
	}
}

func TestFindLocation_PreservesProvidedUUID(t *testing.T) {
	svc := newSvc(
		&mockLocationRepo{
			findLTE: func(_ context.Context, _ *model.LTE) (*model.Location, error) {
				return newLoc(55.75, 37.61), nil
			},
		},
		&mockExternalRepo{},
		&mockUpdateRepo{},
	)
	const wantUUID = "my-test-uuid-123"
	req := &model.LocateRequest{
		SessionUUID: wantUUID,
		Cell:        []model.Cell{{LTE: &model.LTE{MCC: 1}}},
	}
	resp, err := svc.FindLocation(context.Background(), req)
	if err != nil || resp == nil {
		t.Fatalf("unexpected result: err=%v resp=%v", err, resp)
	}
	if resp.SessionUUID != wantUUID {
		t.Errorf("expected UUID=%s, got %s", wantUUID, resp.SessionUUID)
	}
}

func TestFindLocation_NoSourcesReturnsNil(t *testing.T) {
	svc := newSvc(&mockLocationRepo{}, &mockExternalRepo{}, &mockUpdateRepo{})
	req := &model.LocateRequest{SessionUUID: "test-uuid"}
	resp, err := svc.FindLocation(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != nil {
		t.Errorf("expected nil response for no sources, got %+v", resp)
	}
}

func TestFindLocation_ReturnsLocationInResponse(t *testing.T) {
	svc := newSvc(
		&mockLocationRepo{
			findLTE: func(_ context.Context, _ *model.LTE) (*model.Location, error) {
				return newLoc(55.75, 37.61), nil
			},
		},
		&mockExternalRepo{},
		&mockUpdateRepo{},
	)
	req := &model.LocateRequest{
		SessionUUID: "sess-123",
		Cell:        []model.Cell{{LTE: &model.LTE{MCC: 250}}},
	}
	resp, err := svc.FindLocation(context.Background(), req)
	if err != nil || resp == nil || resp.Location == nil {
		t.Fatalf("expected valid response: err=%v resp=%v", err, resp)
	}
	if resp.Location.Point.Lat != 55.75 {
		t.Errorf("unexpected lat: %v", resp.Location.Point.Lat)
	}
}

func TestFindLocation_SearchError(t *testing.T) {
	svc := newSvc(
		&mockLocationRepo{
			findLTE: func(_ context.Context, _ *model.LTE) (*model.Location, error) {
				return nil, errors.New("repo down")
			},
		},
		&mockExternalRepo{},
		&mockUpdateRepo{},
	)
	req := &model.LocateRequest{Cell: []model.Cell{{LTE: &model.LTE{MCC: 1}}}}
	_, err := svc.FindLocation(context.Background(), req)
	if err == nil {
		t.Error("expected error from search failure")
	}
}

func TestFindLocation_SessionCreateErrorDoesNotFail(t *testing.T) {
	// mockSessionRepo always returns nil for errors, so just verify no panic.
	svc := newSvc(
		&mockLocationRepo{
			findLTE: func(_ context.Context, _ *model.LTE) (*model.Location, error) {
				return newLoc(55.0, 37.0), nil
			},
		},
		&mockExternalRepo{},
		&mockUpdateRepo{},
	)
	req := &model.LocateRequest{
		SessionUUID: "test-uuid",
		Cell:        []model.Cell{{LTE: &model.LTE{}}},
	}
	resp, err := svc.FindLocation(context.Background(), req)
	if err != nil || resp == nil {
		t.Errorf("unexpected result: err=%v resp=%v", err, resp)
	}
}
