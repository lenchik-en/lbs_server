package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/lenchik-en/lbs_server/internal/domain/model"
)

// captureSessionRepo records ListSessions call parameters for assertion.
type captureSessionRepo struct {
	capturedLimit  int
	capturedOffset int
	rows           []model.SessionListRow
	total          int64
	listErr        error
	points         []model.SessionPoint
	pointsErr      error
}

func (r *captureSessionRepo) CreateIfNotExists(_ context.Context, _ string) error { return nil }
func (r *captureSessionRepo) SavePoint(_ context.Context, _ string, _, _ float64, _ int, _ string, _, _ any) error {
	return nil
}

func (r *captureSessionRepo) ListSessions(_ context.Context, limit, offset int) ([]model.SessionListRow, int64, error) {
	r.capturedLimit = limit
	r.capturedOffset = offset
	return r.rows, r.total, r.listErr
}

func (r *captureSessionRepo) GetSessionPoints(_ context.Context, _ string) ([]model.SessionPoint, error) {
	return r.points, r.pointsErr
}
func (r *captureSessionRepo) UpdatePoint(_ context.Context, _ int64, _, _ float64, _ int) error {
	return nil
}

// --- List: pagination ---

func TestSessionList_ZeroPageNormalized(t *testing.T) {
	repo := &captureSessionRepo{}
	svc := NewSessionAdminService(repo, &mockRefreshDest{})
	svc.List(context.Background(), 0)
	if repo.capturedOffset != 0 {
		t.Errorf("page=0 should normalize to page=1 (offset=0), got offset=%d", repo.capturedOffset)
	}
	if repo.capturedLimit != SessionPageSize {
		t.Errorf("expected limit=%d, got %d", SessionPageSize, repo.capturedLimit)
	}
}

func TestSessionList_NegativePageNormalized(t *testing.T) {
	repo := &captureSessionRepo{}
	svc := NewSessionAdminService(repo, &mockRefreshDest{})
	svc.List(context.Background(), -5)
	if repo.capturedOffset != 0 {
		t.Errorf("page=-5 should normalize to page=1 (offset=0), got offset=%d", repo.capturedOffset)
	}
}

func TestSessionList_Page1Offset0(t *testing.T) {
	repo := &captureSessionRepo{}
	svc := NewSessionAdminService(repo, &mockRefreshDest{})
	svc.List(context.Background(), 1)
	if repo.capturedOffset != 0 {
		t.Errorf("expected offset=0 for page=1, got %d", repo.capturedOffset)
	}
}

func TestSessionList_Page2Offset50(t *testing.T) {
	repo := &captureSessionRepo{}
	svc := NewSessionAdminService(repo, &mockRefreshDest{})
	svc.List(context.Background(), 2)
	want := SessionPageSize
	if repo.capturedOffset != want {
		t.Errorf("expected offset=%d for page=2, got %d", want, repo.capturedOffset)
	}
}

func TestSessionList_Page3Offset100(t *testing.T) {
	repo := &captureSessionRepo{}
	svc := NewSessionAdminService(repo, &mockRefreshDest{})
	svc.List(context.Background(), 3)
	want := 2 * SessionPageSize
	if repo.capturedOffset != want {
		t.Errorf("expected offset=%d for page=3, got %d", want, repo.capturedOffset)
	}
}

func TestSessionList_ReturnsRowsAndTotal(t *testing.T) {
	now := time.Now()
	rows := []model.SessionListRow{
		{SessionUUID: "uuid1", PointCount: 5, StartedAt: now},
		{SessionUUID: "uuid2", PointCount: 3, StartedAt: now},
	}
	repo := &captureSessionRepo{rows: rows, total: 42}
	svc := NewSessionAdminService(repo, &mockRefreshDest{})
	got, total, err := svc.List(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 42 {
		t.Errorf("expected total=42, got %d", total)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 rows, got %d", len(got))
	}
}

func TestSessionList_RepoError(t *testing.T) {
	repo := &captureSessionRepo{listErr: errors.New("db error")}
	svc := NewSessionAdminService(repo, &mockRefreshDest{})
	_, _, err := svc.List(context.Background(), 1)
	if err == nil {
		t.Error("expected error from repo")
	}
}

// --- GetPoints ---

func TestGetPoints_ReturnsPoints(t *testing.T) {
	now := time.Now()
	points := []model.SessionPoint{
		{ID: 1, Timestamp: now, Lat: 55.75, Lon: 37.61, Accuracy: 100},
		{ID: 2, Timestamp: now, Lat: 55.76, Lon: 37.62, Accuracy: 50},
	}
	repo := &captureSessionRepo{points: points}
	svc := NewSessionAdminService(repo, &mockRefreshDest{})
	got, err := svc.GetPoints(context.Background(), "test-uuid")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 points, got %d", len(got))
	}
}

func TestGetPoints_EmptyResult(t *testing.T) {
	repo := &captureSessionRepo{points: nil}
	svc := NewSessionAdminService(repo, &mockRefreshDest{})
	got, err := svc.GetPoints(context.Background(), "unknown-uuid")
	if err != nil || got != nil {
		t.Errorf("expected nil,nil; got %v err=%v", got, err)
	}
}

func TestGetPoints_RepoError(t *testing.T) {
	repo := &captureSessionRepo{pointsErr: errors.New("db error")}
	svc := NewSessionAdminService(repo, &mockRefreshDest{})
	_, err := svc.GetPoints(context.Background(), "test-uuid")
	if err == nil {
		t.Error("expected error from repo")
	}
}
