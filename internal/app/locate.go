package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/lenchik-en/lbs_server/internal/db"
	"github.com/lenchik-en/lbs_server/internal/models"
)

func (a *App) HandleLocate(w http.ResponseWriter, r *http.Request) {
	log.Printf("[INFO] get /locate request from client %s", r.RemoteAddr)
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.LocateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	//TODO: уточнить нужно ли присваивать, если не получен от клиента(или генерирует со стороны клиента)
	if req.SessionUUID == "" {
		req.SessionUUID = uuid.New().String()
	}
	log.Printf("SessionUUID: %s", req.SessionUUID)

	if err := a.session.CreateSessionIfNotExists(r.Context(), req.SessionUUID); err != nil {
		log.Printf("[WARN] failed to create session %s: %v", req.SessionUUID, err)
	}

	var location *models.Location
	for _, cell := range req.Cell {
		loc, err := a.findLocation(r.Context(), cell)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to find location: %v", err), http.StatusInternalServerError)
			return
		}
		if loc != nil {
			location = loc
			//На этапе прототипа поиск координат прекращается при нахождении первой валидной точки, что позволяет
			//минимизировать задержку ответа и количество обращений к источникам данных.
			//В дальнейшем возможно расширение алгоритма до агрегации нескольких источников.
			break
		}

	}

	if location == nil {
		http.Error(w, "Location not found", http.StatusNotFound)
		return
	}

	if err := a.session.SavePoint(r.Context(), req.SessionUUID, location.Point.Lat, location.Point.Lon, location.Accuracy, "external", req, location); err != nil {
		log.Printf("[WARN] failed to save point for session %s: %v", req.SessionUUID, err)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{
		"sessionUUID": req.SessionUUID,
		"location":    location,
	}); err != nil {
		log.Printf("[WARN] failed to encode response: %v", err)
	}
	log.Printf("POST /locate for Client %s is done", r.RemoteAddr)
}

func (a *App) findLocation(ctx context.Context, cell models.Cell) (*models.Location, error) {
	//1. Looking at LocateDB
	loc, err := a.findInDB(ctx, a.locateDB, cell)
	if err != nil {
		return nil, fmt.Errorf("error in locateDB: %v", err)
	}

	if loc != nil {
		return loc, nil
	}

	//2. If not in LocateDB, then looking at ExternalDB
	loc, err = a.findInDB(ctx, a.externalDB, cell)
	if err != nil {
		return nil, fmt.Errorf("error in externalDB: %v", err)
	}

	//3. if found, then save it to UpdateDB(TODO: or LocateDB)
	if loc != nil {
		go func() {
			//_ = a.locateDB.SaveLTE
		}()
	}
	return loc, nil
}

func (a *App) findInDB(ctx context.Context, finder db.CellFinder, cell models.Cell) (*models.Location, error) {
	switch {
	case cell.LTE != nil:
		return finder.FindLTE(ctx, cell.LTE)
	case cell.GSM != nil:
		return finder.FindGSM(ctx, cell.GSM)
	case cell.WCDMA != nil:
		return finder.FindWCDMA(ctx, cell.WCDMA)
	default:
		return nil, fmt.Errorf("unknown type of radio")
	}
}
