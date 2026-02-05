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

	session := a.session
	err := session.CreateSessionIfNotExists(r.Context(), req.SessionUUID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to add sessionUUID to the database: %v", err), http.StatusInternalServerError)
		//return
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

	_ = session.SavePoint(r.Context(), req.SessionUUID, location.Point.Lat, location.Point.Lon, location.Accuracy, "opencell", req, location)

	out := map[string]any{
		"sessionUUID": req.SessionUUID,
		"location":    location,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
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
		//_ = a.locateDB.SaveLTE
		return loc, nil
	}

	return nil, nil
}

func (a *App) findInDB(ctx context.Context, db db.CellFinder, cell models.Cell) (*models.Location, error) {
	switch {
	case cell.LTE != nil:
		return db.FindLTE(ctx, cell.LTE)
	case cell.GSM != nil:
		return db.FindGSM(ctx, cell.GSM)
	case cell.WCDMA != nil:
		return db.FindWCDMA(ctx, cell.WCDMA)
	default:
		return nil, fmt.Errorf("unknown type of radio")
	}
}
