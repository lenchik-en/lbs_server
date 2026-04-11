package app

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lenchik-en/lbs_server/internal/config"
	"github.com/lenchik-en/lbs_server/internal/models"
	"github.com/lenchik-en/lbs_server/internal/ratelimit"
)

// newTestApp создаёт App с нулевым rate limit (все запросы проходят)
// и подменёнными моками БД.
func newTestApp(locate *mockFinder, external *mockCellFinder, update *mockUpdateDB) *App {
	return &App{
		locateDB:   locate,
		externalDB: external,
		updateDB:   update,
		session:    &mockSession{},
		cfg: &config.Config{
			// 0 = rate limit отключён
			LocateMinPeriodKey:     0,
			LocateMinPeriodUUID:    0,
			LocateMinPeriodKeyUUID: 0,
			UpdateMinPeriodKey:     0,
			UpdateMinPeriodUUID:    0,
			UpdateMinPeriodKeyUUID: 0,
		},
		rateLimiter: ratelimit.New(),
	}
}

// --- HandleLocate ---

func TestHandleLocate_MethodNotAllowed(t *testing.T) {
	a := newTestApp(&mockFinder{}, &mockCellFinder{}, &mockUpdateDB{})
	defer a.rateLimiter.Stop()

	for _, method := range []string{http.MethodGet, http.MethodPut, http.MethodDelete, http.MethodPatch} {
		req := httptest.NewRequest(method, "/locate", nil)
		w := httptest.NewRecorder()
		a.HandleLocate(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("method %s: expected 405, got %d", method, w.Code)
		}
	}
}

func TestHandleLocate_InvalidJSON(t *testing.T) {
	a := newTestApp(&mockFinder{}, &mockCellFinder{}, &mockUpdateDB{})
	defer a.rateLimiter.Stop()

	req := httptest.NewRequest(http.MethodPost, "/locate", strings.NewReader("{bad json"))
	w := httptest.NewRecorder()
	a.HandleLocate(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleLocate_NothingFound_EmptyBody(t *testing.T) {
	// Ничего не найдено → resp == nil → handler возвращает 200 с пустым телом
	a := newTestApp(&mockFinder{}, &mockCellFinder{}, &mockUpdateDB{})
	defer a.rateLimiter.Stop()

	body := models.LocateRequest{
		Cell: []models.Cell{{LTE: &models.LTE{MCC: 250}}},
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/locate", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()
	a.HandleLocate(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 when nothing found, got %d", w.Code)
	}
	if w.Body.Len() != 0 {
		t.Errorf("expected empty body when nothing found, got %q", w.Body.String())
	}
}

func TestHandleLocate_FoundCell_ResponseShape(t *testing.T) {
	locate := &mockFinder{
		mockCellFinder: mockCellFinder{
			findLTE: func(_ context.Context, _ *models.LTE) (*models.Location, error) {
				return newLoc(55.75, 37.61), nil
			},
		},
	}

	a := newTestApp(locate, &mockCellFinder{}, &mockUpdateDB{})
	defer a.rateLimiter.Stop()

	body := models.LocateRequest{
		Cell: []models.Cell{{LTE: &models.LTE{MCC: 250, MNC: 99, TAC: 1000, CI: 5632016}}},
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/locate", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()
	a.HandleLocate(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", w.Code, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}

	var resp LocateResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Location == nil {
		t.Fatal("expected location in response")
	}
	if resp.Location.Point.Lat != 55.75 {
		t.Errorf("expected lat=55.75, got %v", resp.Location.Point.Lat)
	}
	if resp.Location.Accuracy != int(defaultCellAccuracy) {
		t.Errorf("expected accuracy=%d, got %d", int(defaultCellAccuracy), resp.Location.Accuracy)
	}
	if resp.SessionUUID == "" {
		t.Error("expected non-empty SessionUUID in response")
	}
}

func TestHandleLocate_SessionUUIDPassedThrough(t *testing.T) {
	locate := &mockFinder{
		mockCellFinder: mockCellFinder{
			findLTE: func(_ context.Context, _ *models.LTE) (*models.Location, error) {
				return newLoc(55.75, 37.61), nil
			},
		},
	}

	a := newTestApp(locate, &mockCellFinder{}, &mockUpdateDB{})
	defer a.rateLimiter.Stop()

	const wantUUID = "my-session-uuid-123"
	body := models.LocateRequest{
		SessionUUID: wantUUID,
		Cell:        []models.Cell{{LTE: &models.LTE{MCC: 250}}},
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/locate", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()
	a.HandleLocate(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp LocateResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.SessionUUID != wantUUID {
		t.Errorf("expected sessionUUID=%q, got %q", wantUUID, resp.SessionUUID)
	}
}

func TestHandleLocate_RateLimitExceeded(t *testing.T) {
	a := newTestApp(&mockFinder{}, &mockCellFinder{}, &mockUpdateDB{})
	a.cfg.LocateMinPeriodUUID = 60_000 // 60 секунд по UUID
	defer a.rateLimiter.Stop()

	makeReq := func() *httptest.ResponseRecorder {
		body := models.LocateRequest{
			SessionUUID: "fixed-uuid",
			Cell:        []models.Cell{{LTE: &models.LTE{MCC: 250}}},
		}
		bodyBytes, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/locate", bytes.NewReader(bodyBytes))
		w := httptest.NewRecorder()
		a.HandleLocate(w, req)
		return w
	}

	first := makeReq()
	if first.Code != http.StatusOK {
		t.Fatalf("first request must succeed, got %d", first.Code)
	}

	second := makeReq()
	if second.Code != http.StatusTooManyRequests {
		t.Errorf("second request within period must be 429, got %d", second.Code)
	}
}

// --- HandleUpdate ---

func TestHandleUpdate_MethodNotAllowed(t *testing.T) {
	a := newTestApp(&mockFinder{}, &mockCellFinder{}, &mockUpdateDB{})
	defer a.rateLimiter.Stop()

	for _, method := range []string{http.MethodGet, http.MethodPut, http.MethodDelete} {
		req := httptest.NewRequest(method, "/update", nil)
		w := httptest.NewRecorder()
		a.HandleUpdate(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("method %s: expected 405, got %d", method, w.Code)
		}
	}
}

func TestHandleUpdate_InvalidJSON(t *testing.T) {
	a := newTestApp(&mockFinder{}, &mockCellFinder{}, &mockUpdateDB{})
	defer a.rateLimiter.Stop()

	req := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader("not json"))
	w := httptest.NewRecorder()
	a.HandleUpdate(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleUpdate_NoLocation(t *testing.T) {
	a := newTestApp(&mockFinder{}, &mockCellFinder{}, &mockUpdateDB{})
	defer a.rateLimiter.Stop()

	body := models.UpdateRequest{
		Cell: []models.Cell{{LTE: &models.LTE{MCC: 250}}},
		// Location не задан
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()
	a.HandleUpdate(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing location, got %d", w.Code)
	}
}

func TestHandleUpdate_NoSource(t *testing.T) {
	a := newTestApp(&mockFinder{}, &mockCellFinder{}, &mockUpdateDB{})
	defer a.rateLimiter.Stop()

	body := models.UpdateRequest{
		Location: &models.Location{Point: models.Point{Lat: 55.75, Lon: 37.61}, Accuracy: 50},
		// нет cell, wifi, ip
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()
	a.HandleUpdate(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for no source, got %d", w.Code)
	}
}

func TestHandleUpdate_ValidRequest(t *testing.T) {
	a := newTestApp(&mockFinder{}, &mockCellFinder{}, &mockUpdateDB{})
	defer a.rateLimiter.Stop()

	body := models.UpdateRequest{
		Cell:     []models.Cell{{LTE: &models.LTE{MCC: 250, MNC: 99, TAC: 1000, CI: 5632016}}},
		Location: &models.Location{Point: models.Point{Lat: 55.75, Lon: 37.61}, Accuracy: 50},
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()
	a.HandleUpdate(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d (body: %s)", w.Code, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}
}

func TestHandleUpdate_InvalidObjectType(t *testing.T) {
	a := newTestApp(&mockFinder{}, &mockCellFinder{}, &mockUpdateDB{})
	defer a.rateLimiter.Stop()

	body := models.UpdateRequest{
		Cell:     []models.Cell{{LTE: &models.LTE{MCC: 250}}},
		Location: &models.Location{Point: models.Point{Lat: 55.75, Lon: 37.61}, Accuracy: 50},
		Type:     "garage",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()
	a.HandleUpdate(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid type, got %d", w.Code)
	}
}

func TestHandleUpdate_InvalidLatitude(t *testing.T) {
	a := newTestApp(&mockFinder{}, &mockCellFinder{}, &mockUpdateDB{})
	defer a.rateLimiter.Stop()

	body := models.UpdateRequest{
		Cell:     []models.Cell{{LTE: &models.LTE{MCC: 250}}},
		Location: &models.Location{Point: models.Point{Lat: 200, Lon: 37.61}, Accuracy: 50},
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()
	a.HandleUpdate(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid lat, got %d", w.Code)
	}
}

func TestHandleUpdate_RateLimitExceeded(t *testing.T) {
	a := newTestApp(&mockFinder{}, &mockCellFinder{}, &mockUpdateDB{})
	a.cfg.UpdateMinPeriodKey = 60_000 // 60 секунд по API-ключу
	defer a.rateLimiter.Stop()

	makeReq := func() *httptest.ResponseRecorder {
		body := models.UpdateRequest{
			Cell:     []models.Cell{{LTE: &models.LTE{MCC: 250}}},
			Location: &models.Location{Point: models.Point{Lat: 55.75, Lon: 37.61}, Accuracy: 50},
		}
		bodyBytes, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(bodyBytes))
		req.Header.Set("X-Api-Key", "test-api-key")
		w := httptest.NewRecorder()
		a.HandleUpdate(w, req)
		return w
	}

	first := makeReq()
	if first.Code != http.StatusOK {
		t.Fatalf("first request must succeed, got %d", first.Code)
	}

	second := makeReq()
	if second.Code != http.StatusTooManyRequests {
		t.Errorf("second request within period must be 429, got %d", second.Code)
	}
}

func TestHandleUpdate_WithSessionUUID_OK(t *testing.T) {
	a := newTestApp(&mockFinder{}, &mockCellFinder{}, &mockUpdateDB{})
	defer a.rateLimiter.Stop()

	body := models.UpdateRequest{
		SessionUUID: "session-abc",
		Cell:        []models.Cell{{LTE: &models.LTE{MCC: 250}}},
		Location:    &models.Location{Point: models.Point{Lat: 55.75, Lon: 37.61}, Accuracy: 50},
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()
	a.HandleUpdate(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d (body: %s)", w.Code, w.Body.String())
	}
}

// --- HandleHealth ---

func TestHandleHealth(t *testing.T) {
	a := &App{}
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	a.HandleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode health response: %v", err)
	}
	if resp["status"] != "ok" {
		t.Errorf("expected status=ok, got %q", resp["status"])
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}
}
