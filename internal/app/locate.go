package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/lenchik-en/lbs_server/internal/db"
	"github.com/lenchik-en/lbs_server/internal/geo"
	"github.com/lenchik-en/lbs_server/internal/models"
)

// Дефолтные значения accuracy todo: вынести в конфиг.
const (
	defaultCellAccuracy    = 1000.0
	defaultWifiAccuracy    = 100.0
	defaultIPAccuracy      = 100.0
	defaultWifiMaxDistance = 5000.0
	defaultMinAccuracy     = 1.0
)

type LocateResponse struct {
	SessionUUID string           `json:"sessionUUID"`
	Location    *models.Location `json:"location"`
}

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

	resp, err := a.findLocation(r.Context(), &req)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to find location: %v", err), http.StatusInternalServerError)
		return
	}

	if resp == nil {
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("[WARN] failed to encode response: %v", err)
	}
	log.Printf("[INFO] POST /locate for Client %s is done", r.RemoteAddr)
}

func (a *App) findLocation(ctx context.Context, req *models.LocateRequest) (*LocateResponse, error) {
	if req.SessionUUID == "" {
		req.SessionUUID = uuid.New().String()
	}
	log.Printf("[INFO] SessionUUID: %s", req.SessionUUID)

	if err := a.session.CreateSessionIfNotExists(ctx, req.SessionUUID); err != nil {
		log.Printf("[WARN] failed to create session %s: %v", req.SessionUUID, err)
	}

	location, err := a.searchBySource(ctx, req)
	if err != nil {
		return nil, err
	}
	if location == nil {
		return nil, nil
	}

	if err := a.session.SavePoint(ctx, req.SessionUUID, location.Point.Lat, location.Point.Lon, location.Accuracy, models.EXTERNAL, *req, location); err != nil {
		log.Printf("[WARN] failed to save point for session %s: %v", req.SessionUUID, err)
	}

	return &LocateResponse{
		SessionUUID: req.SessionUUID,
		Location:    location,
	}, nil
}

// searchBySource реализует алгоритм выбора локации:
//  1. Ищем первую найденную Cell-координату
//  2. Ищем первую найденную WiFi-координату
//  3. Если оба найдены и расстояние < порога — WiFi подтверждает Cell,
//     accuracy вычисляется по алгоритму хорды пересечения окружностей
//  4. Иначе — fallback по приоритету Cell → WiFi → IP
func (a *App) searchBySource(ctx context.Context, req *models.LocateRequest) (*models.Location, error) {
	cellLoc, err := a.findCell(ctx, req.Cell)
	if err != nil {
		return nil, fmt.Errorf("cell: %w", err)
	}

	wifiLoc, err := a.findWifi(ctx, req.Wifi)
	if err != nil {
		return nil, fmt.Errorf("wifi: %w", err)
	}

	// WiFi подтверждает Cell?
	if cellLoc != nil && wifiLoc != nil {
		d := geo.HaversineDistance(cellLoc.Lat, cellLoc.Lon, wifiLoc.Lat, wifiLoc.Lon)
		if d < defaultWifiMaxDistance {
			accuracy := geo.ChordAccuracy(d, defaultCellAccuracy, defaultWifiAccuracy, defaultMinAccuracy)
			wifiLoc.Accuracy = int(accuracy)
			log.Printf("[INFO] WiFi confirms Cell (distance=%.0fm, accuracy=%dm)", d, wifiLoc.Accuracy)
			return wifiLoc, nil
		}
		log.Printf("[INFO] WiFi too far from Cell (distance=%.0fm > %.0fm), using Cell only", d, defaultWifiMaxDistance)
	}

	if cellLoc != nil {
		cellLoc.Accuracy = int(defaultCellAccuracy)
		return cellLoc, nil
	}

	if wifiLoc != nil {
		wifiLoc.Accuracy = int(defaultWifiAccuracy)
		return wifiLoc, nil
	}

	//если ничего больше не осталось, то берем IP
	ipLoc, err := a.findIP(ctx, req.IP)
	if err != nil {
		return nil, fmt.Errorf("ip: %w", err)
	}
	if ipLoc != nil {
		ipLoc.Accuracy = int(defaultIPAccuracy)
		return ipLoc, nil
	}

	return nil, nil
}

func (a *App) findCell(ctx context.Context, cells []models.Cell) (*models.Location, error) {
	for _, cell := range cells {
		loc, err := a.findCellLocation(ctx, cell)
		if err != nil {
			return nil, err
		}
		if loc != nil {
			return loc, nil
		}
	}
	return nil, nil
}

func (a *App) findWifi(ctx context.Context, wifis []models.Wifi) (*models.Location, error) {
	for _, wifi := range wifis {
		loc, err := a.locateDB.FindWifi(ctx, &wifi)
		if err != nil {
			return nil, err
		}
		if loc != nil {
			return loc, nil
		}
	}
	return nil, nil
}

func (a *App) findIP(ctx context.Context, ips []models.Ip) (*models.Location, error) {
	for _, ip := range ips {
		loc, err := a.locateDB.FindIP(ctx, &ip)
		if err != nil {
			return nil, err
		}
		if loc != nil {
			return loc, nil
		}
	}
	return nil, nil
}

func (a *App) findCellLocation(ctx context.Context, cell models.Cell) (*models.Location, error) {
	// 1. LocateDB
	loc, err := findCellInDB(ctx, a.locateDB, cell)
	if err != nil {
		return nil, fmt.Errorf("locateDB: %w", err)
	}
	if loc != nil {
		log.Println("[INFO] location found in LocateDB")
		return loc, nil
	}

	// 2. ExternalDB
	loc, err = findCellInDB(ctx, a.externalDB, cell)
	if err != nil {
		return nil, fmt.Errorf("externalDB: %w", err)
	}

	// 3. Save to UpdateDB for enrichment
	if loc != nil {
		log.Println("[INFO] location found in ExternalDB")
		cellCopy := cell
		locCopy := loc
		go func() {
			if err := a.updateDB.InsertCell(context.Background(), &cellCopy, locCopy, models.EXTERNAL, nil); err != nil {
				log.Printf("[WARN] failed to save cell from ExternalDB to UpdateDB: %v", err)
			}
			log.Println("[INFO] save cell from ExternalDB to UpdateDB")
		}()
	}
	return loc, nil
}

func findCellInDB(ctx context.Context, finder db.CellFinder, cell models.Cell) (*models.Location, error) {
	switch {
	case cell.LTE != nil:
		return finder.FindLTE(ctx, cell.LTE)
	case cell.GSM != nil:
		return finder.FindGSM(ctx, cell.GSM)
	case cell.WCDMA != nil:
		return finder.FindWCDMA(ctx, cell.WCDMA)
	default:
		return nil, fmt.Errorf("cell: unknown radio type")
	}
}
