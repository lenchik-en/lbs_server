package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"

	"github.com/lenchik-en/lbs_server/internal/models"
)

func (a *App) HandleUpdate(w http.ResponseWriter, r *http.Request) {
	log.Printf("[INFO] get /update request from client %s", r.RemoteAddr)
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	reqBody, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	var req models.UpdateRequest
	if err := json.Unmarshal(reqBody, &req); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if err := a.validateUpdate(req); err != nil {
		http.Error(w, fmt.Sprintf("Validation error: %v", err), http.StatusBadRequest)
		return
	}

	// Если клиент передал sessionUUID — привязываем точку к сессии
	if req.SessionUUID != "" {
		if err := a.session.CreateSessionIfNotExists(r.Context(), req.SessionUUID); err != nil {
			log.Printf("[WARN] failed to create session %s: %v", req.SessionUUID, err)
		}
		if err := a.session.SavePoint(r.Context(), req.SessionUUID, req.Location.Point.Lat, req.Location.Point.Lon, req.Location.Accuracy, "client", req, nil); err != nil {
			log.Printf("[WARN] failed to save point for session %s: %v", req.SessionUUID, err)
		}
	}

	if err := a.insertToUpdate(r.Context(), req, reqBody); err != nil {
		http.Error(w, "Failed to insert to UpdateDB", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode("Location is added to UpdateDB"); err != nil {
		http.Error(w, fmt.Sprintf("failed to encode response: %v", err), http.StatusInternalServerError)
		return
	}
	log.Printf("[INFO] POST /update for Client %s is done", r.RemoteAddr)
}

func (a *App) insertToUpdate(ctx context.Context, req models.UpdateRequest, rawJSON any) error {
	for i := range req.Cell {
		if err := a.updateDB.InsertCell(ctx, &req.Cell[i], req.Location, models.CLIENT, rawJSON); err != nil {
			return fmt.Errorf("failed to insert cell[%d]: %w", i, err)
		}
	}

	for i := range req.Wifi {
		if err := a.updateDB.InsertWifi(ctx, &req.Wifi[i], models.CLIENT, rawJSON); err != nil {
			return fmt.Errorf("failed to insert wifi[%d]: %w", i, err)
		}
	}

	for i := range req.IP {
		if err := a.updateDB.InsertIP(ctx, &req.IP[i], models.CLIENT, rawJSON); err != nil {
			return fmt.Errorf("failed to insert ip[%d]: %w", i, err)
		}
	}

	return nil
}

func (a *App) validateUpdate(req models.UpdateRequest) error {
	if req.Location == nil {
		return fmt.Errorf("no location provided")
	}
	if err := a.validateLocation(req.Location); err != nil {
		return fmt.Errorf("invalid location: %v", err)
	}

	hasSource := len(req.Cell) > 0 || len(req.Wifi) > 0 || len(req.IP) > 0
	if !hasSource {
		return fmt.Errorf("at least one source (cell, wifi or ip) must be provided")
	}

	for i, cell := range req.Cell {
		if err := validateCell(i, cell); err != nil {
			return err
		}
	}

	for i, ip := range req.IP {
		if err := validateIP(i, ip); err != nil {
			return fmt.Errorf("validateIP err: %v", err)
		}
	}

	return nil
}

func validateCell(i int, cell models.Cell) error {
	if cell.LTE == nil && cell.GSM == nil && cell.WCDMA == nil {
		return fmt.Errorf("cell[%d]: no radio type specified (lte, gsm or wcdma required)", i)
	}
	return nil
}

func (a *App) validateLocation(loc *models.Location) error {
	if loc.Point.Lat < -90 || loc.Point.Lat > 90 {
		return fmt.Errorf("latitude out of range")
	}
	if loc.Point.Lon < -180 || loc.Point.Lon > 180 {
		return fmt.Errorf("longitude out of range")
	}
	if loc.Accuracy <= 0 {
		return fmt.Errorf("accuracy must be positive")
	}
	return nil
}

func validateIP(i int, ip models.Ip) error {
	p := fmt.Sprintf("ip[%d]", i)
	if ip.Address == "" {
		return fmt.Errorf("%s: address is required", p)
	}
	if net.ParseIP(ip.Address) == nil {
		return fmt.Errorf("%s: invalid ip address", p)
	}
	return nil
}
