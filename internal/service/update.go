package service

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/lenchik-en/lbs_server/internal/domain/model"
	"github.com/lenchik-en/lbs_server/internal/domain/port"
)

type UpdateService struct {
	updateRepo  port.UpdateRepository
	sessionRepo port.SessionRepository
}

func NewUpdateService(updateRepo port.UpdateRepository, sessionRepo port.SessionRepository) *UpdateService {
	return &UpdateService{updateRepo: updateRepo, sessionRepo: sessionRepo}
}

func (s *UpdateService) Process(ctx context.Context, req *model.UpdateRequest, rawJSON any) error {
	if req.SessionUUID != "" {
		if err := s.sessionRepo.CreateIfNotExists(ctx, req.SessionUUID); err != nil {
			log.Printf("[WARN] failed to create session %s: %v", req.SessionUUID, err)
		}
		if err := s.sessionRepo.SavePoint(ctx, req.SessionUUID,
			req.Location.Point.Lat, req.Location.Point.Lon, req.Location.Accuracy,
			"client", req, nil,
		); err != nil {
			log.Printf("[WARN] failed to save point for session %s: %v", req.SessionUUID, err)
		}
	}

	return s.insert(ctx, req, rawJSON)
}

func (s *UpdateService) insert(ctx context.Context, req *model.UpdateRequest, rawJSON any) error {
	objType := string(req.Type)

	for i := range req.Cell {
		if err := s.updateRepo.InsertCell(ctx, &req.Cell[i], req.Location, "client", objType, rawJSON); err != nil {
			return fmt.Errorf("insert cell[%d]: %w", i, err)
		}
	}
	for i := range req.Wifi {
		if err := s.updateRepo.InsertWifi(ctx, &req.Wifi[i], req.Location, "client", objType, rawJSON); err != nil {
			return fmt.Errorf("insert wifi[%d]: %w", i, err)
		}
	}
	for i := range req.IP {
		if err := s.updateRepo.InsertIP(ctx, &req.IP[i], req.Location, "client", objType, rawJSON); err != nil {
			return fmt.Errorf("insert ip[%d]: %w", i, err)
		}
	}
	return nil
}

func (s *UpdateService) Validate(req *model.UpdateRequest) error {
	if req.Location == nil {
		return fmt.Errorf("no location provided")
	}
	if err := validateLocation(req.Location); err != nil {
		return fmt.Errorf("invalid location: %w", err)
	}
	if req.Type != "" {
		switch req.Type {
		case model.ObjectTypeMetro, model.ObjectTypeBuilding, model.ObjectTypeStreet:
		default:
			return fmt.Errorf("invalid type %q: must be metro, building or street", req.Type)
		}
	}
	if len(req.Cell) == 0 && len(req.Wifi) == 0 && len(req.IP) == 0 {
		return fmt.Errorf("at least one source (cell, wifi or ip) must be provided")
	}
	for i, cell := range req.Cell {
		if cell.LTE == nil && cell.GSM == nil && cell.WCDMA == nil {
			return fmt.Errorf("cell[%d]: no radio type specified", i)
		}
	}
	for i, ip := range req.IP {
		if ip.Address == "" {
			return fmt.Errorf("ip[%d]: address required", i)
		}
		if net.ParseIP(ip.Address) == nil {
			return fmt.Errorf("ip[%d]: invalid address", i)
		}
	}
	return nil
}

func validateLocation(loc *model.Location) error {
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
