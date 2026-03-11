package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
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

	if err := a.insertToUpdate(r.Context(), req, reqBody); err != nil {
		http.Error(w, "Failed to insert to UpdateDB", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode("Location is added to UpdateDB"); err != nil {
		http.Error(w, fmt.Sprintf("failed to encode response: %v", err), http.StatusInternalServerError)
		return
	}
	log.Printf("POST /update for Client %s is done", r.RemoteAddr)
}

func (a *App) insertToUpdate(ctx context.Context, req models.UpdateRequest, rawJSON any) error {
	if err := a.validateUpdate(req); err != nil {
		return err
	}

	if err := a.updateDB.InsertCell(context.Background(), req.Cell, req.Location, "client", rawJSON); err != nil {
		return err
	}
	return nil
}

func (a *App) validateUpdate(req models.UpdateRequest) error {
	source := 0
	if req.Cell != nil {
		if req.Cell.LTE == nil && req.Cell.GSM == nil && req.Cell.WCDMA == nil {
			return fmt.Errorf("cell source provided but empty")
		}
		source++
	}
	if req.Wifi != nil {
		source++
	}
	if req.IP != nil {
		source++
	}

	if source != 1 {
		return fmt.Errorf("only one source must be provided")
	}

	if req.Location == nil {
		return fmt.Errorf("no location were provided")
	}

	if err := a.validateLocation(req.Location); err != nil {
		return fmt.Errorf("invalid location: %v", err)
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
