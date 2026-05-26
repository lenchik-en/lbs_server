package service

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/lenchik-en/lbs_server/internal/domain/model"
	"github.com/lenchik-en/lbs_server/internal/domain/port"
	"github.com/lenchik-en/lbs_server/internal/geo"
)

const (
	defaultCellAccuracy    = 1000.0
	defaultWifiAccuracy    = 100.0
	defaultIPAccuracy      = 100.0
	defaultWifiMaxDistance = 5000.0
	defaultMinAccuracy     = 1.0
)

type LocateService struct {
	locateRepo   port.LocationRepository
	externalRepo port.ExternalRepository
	updateRepo   port.UpdateRepository
	sessionRepo  port.SessionRepository
	wg           asyncGroup
}

func NewLocateService(
	locateRepo port.LocationRepository,
	externalRepo port.ExternalRepository,
	updateRepo port.UpdateRepository,
	sessionRepo port.SessionRepository,
	wg asyncGroup,
) *LocateService {
	return &LocateService{
		locateRepo:   locateRepo,
		externalRepo: externalRepo,
		updateRepo:   updateRepo,
		sessionRepo:  sessionRepo,
		wg:           wg,
	}
}

func (s *LocateService) FindLocation(ctx context.Context, req *model.LocateRequest) (*model.LocateResponse, error) {
	if req.SessionUUID == "" {
		req.SessionUUID = uuid.New().String()
	}
	log.Printf("[INFO] SessionUUID: %s", req.SessionUUID)

	if err := s.sessionRepo.CreateIfNotExists(ctx, req.SessionUUID); err != nil {
		log.Printf("[WARN] failed to create session %s: %v", req.SessionUUID, err)
	}

	location, err := s.searchBySource(ctx, req)
	if err != nil {
		return nil, err
	}
	if location == nil {
		return nil, nil
	}

	if err := s.sessionRepo.SavePoint(ctx, req.SessionUUID,
		location.Point.Lat, location.Point.Lon, location.Accuracy,
		string(model.SourceExternal), req, location,
	); err != nil {
		log.Printf("[WARN] failed to save point for session %s: %v", req.SessionUUID, err)
	}

	return &model.LocateResponse{SessionUUID: req.SessionUUID, Location: location}, nil
}

func (s *LocateService) searchBySource(ctx context.Context, req *model.LocateRequest) (*model.Location, error) {
	cellLoc, err := s.findCell(ctx, req.Cell)
	if err != nil {
		return nil, fmt.Errorf("cell: %w", err)
	}

	// Вышка из метро — WiFi игнорируем
	if cellLoc != nil && cellLoc.FromMetro {
		cellLoc.Accuracy = float64(defaultCellAccuracy)
		log.Printf("[INFO] metro cell detected, skipping WiFi")
		return cellLoc, nil
	}

	wifiLoc, err := s.findWifi(ctx, req.Wifi)
	if err != nil {
		return nil, fmt.Errorf("wifi: %w", err)
	}

	if cellLoc != nil && wifiLoc != nil {
		d := geo.HaversineDistance(cellLoc.Point.Lat, cellLoc.Point.Lon, wifiLoc.Point.Lat, wifiLoc.Point.Lon)
		if d < defaultWifiMaxDistance {
			accuracy := geo.ChordAccuracy(d, defaultCellAccuracy, defaultWifiAccuracy, defaultMinAccuracy)
			wifiLoc.Accuracy = float64(accuracy)
			log.Printf("[INFO] WiFi confirms Cell (distance=%.0fm, accuracy=%dm)", d, wifiLoc.Accuracy)
			return wifiLoc, nil
		}
		log.Printf("[INFO] WiFi too far from Cell (distance=%.0fm > %.0fm), using Cell only", d, defaultWifiMaxDistance)
	}

	if cellLoc != nil {
		cellLoc.Accuracy = float64(defaultCellAccuracy)
		return cellLoc, nil
	}
	if wifiLoc != nil {
		wifiLoc.Accuracy = float64(defaultWifiAccuracy)
		return wifiLoc, nil
	}

	ipLoc, err := s.findIP(ctx, req.IP)
	if err != nil {
		return nil, fmt.Errorf("ip: %w", err)
	}
	if ipLoc != nil {
		ipLoc.Accuracy = float64(defaultIPAccuracy)
		return ipLoc, nil
	}

	return nil, nil
}

func (s *LocateService) findCell(ctx context.Context, cells []model.Cell) (*model.Location, error) {
	for _, cell := range cells {
		loc, err := s.findCellLocation(ctx, cell)
		if err != nil {
			return nil, err
		}
		if loc != nil {
			return loc, nil
		}
	}
	return nil, nil
}

func (s *LocateService) findWifi(ctx context.Context, wifis []model.Wifi) (*model.Location, error) {
	for _, wifi := range wifis {
		loc, err := s.locateRepo.FindWifi(ctx, &wifi)
		if err != nil {
			return nil, err
		}
		if loc != nil {
			return loc, nil
		}
	}
	return nil, nil
}

func (s *LocateService) findIP(ctx context.Context, ips []model.Ip) (*model.Location, error) {
	for _, ip := range ips {
		loc, err := s.locateRepo.FindIP(ctx, &ip)
		if err != nil {
			return nil, err
		}
		if loc != nil {
			return loc, nil
		}
	}
	return nil, nil
}

func (s *LocateService) findCellLocation(ctx context.Context, cell model.Cell) (*model.Location, error) {
	loc, err := findCellInRepo(ctx, s.locateRepo, cell)
	if err != nil {
		return nil, fmt.Errorf("locateRepo: %w", err)
	}
	if loc != nil {
		log.Println("[INFO] location found in LocateDB")
		return loc, nil
	}

	loc, err = findCellInRepo(ctx, s.externalRepo, cell)
	if err != nil {
		return nil, fmt.Errorf("externalRepo: %w", err)
	}
	if loc != nil {
		log.Println("[INFO] location found in ExternalDB")
		loc.Source = model.SourceExternal
		cellCopy := cell
		locCopy := loc
		s.wg.Go(func() {
			if err := s.updateRepo.InsertCell(context.Background(), &cellCopy, locCopy, string(model.SourceExternal), "", nil); err != nil {
				log.Printf("[WARN] failed to save cell from ExternalDB: %v", err)
			}
		})
	}
	return loc, nil
}

// findCellInRepo ищет ячейку в любом репозитории, реализующем все три метода Find*.
func findCellInRepo(ctx context.Context, repo interface {
	FindLTE(context.Context, *model.LTE) (*model.Location, error)
	FindGSM(context.Context, *model.GSM) (*model.Location, error)
	FindWCDMA(context.Context, *model.WCDMA) (*model.Location, error)
}, cell model.Cell) (*model.Location, error) {
	switch {
	case cell.LTE != nil:
		return repo.FindLTE(ctx, cell.LTE)
	case cell.GSM != nil:
		return repo.FindGSM(ctx, cell.GSM)
	case cell.WCDMA != nil:
		return repo.FindWCDMA(ctx, cell.WCDMA)
	default:
		return nil, fmt.Errorf("unknown radio type")
	}
}
