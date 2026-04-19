package service

import (
	"context"
	"log"
	"time"

	"github.com/lenchik-en/lbs_server/internal/domain/model"
	"github.com/lenchik-en/lbs_server/internal/domain/port"
)

const refreshPageSize = 1000

type RefreshService struct {
	src  port.RefreshSource
	dst  port.RefreshDest
	done chan struct{}
}

func NewRefreshService(src port.RefreshSource, dst port.RefreshDest) *RefreshService {
	return &RefreshService{src: src, dst: dst, done: make(chan struct{})}
}

func (s *RefreshService) Refresh(_ context.Context) (model.RefreshStats, error) {
	var stats model.RefreshStats
	var err error

	stats.CellsUpserted, err = s.refreshCells()
	if err != nil {
		return stats, err
	}
	stats.WifisUpserted, err = s.refreshWifis()
	if err != nil {
		return stats, err
	}
	stats.IPsUpserted, err = s.refreshIPs()
	if err != nil {
		return stats, err
	}

	log.Printf("[INFO] refresh done: cells=%d wifis=%d ips=%d",
		stats.CellsUpserted, stats.WifisUpserted, stats.IPsUpserted)
	return stats, nil
}

func (s *RefreshService) StartAutoRefresh(ctx context.Context, periodMs int) {
	if periodMs == 0 {
		log.Println("[INFO] auto-refresh disabled")
		return
	}
	period := time.Duration(periodMs) * time.Millisecond
	go func() {
		ticker := time.NewTicker(period)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if _, err := s.Refresh(context.Background()); err != nil {
					log.Printf("[WARN] auto-refresh failed: %v", err)
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	log.Printf("[INFO] auto-refresh started (period=%v)", period)
}

func (s *RefreshService) refreshCells() (int, error) {
	total, offset := 0, 0
	for {
		rows, err := s.src.FetchCellsForRefresh(offset, refreshPageSize)
		if err != nil {
			return total, err
		}
		for _, row := range rows {
			if err := s.dst.UpsertCell(row); err != nil {
				log.Printf("[WARN] upsert cell: %v", err)
				continue
			}
			total++
		}
		if len(rows) < refreshPageSize {
			break
		}
		offset += len(rows)
	}
	return total, nil
}

func (s *RefreshService) refreshWifis() (int, error) {
	total, offset := 0, 0
	for {
		rows, err := s.src.FetchWifisForRefresh(offset, refreshPageSize)
		if err != nil {
			return total, err
		}
		for _, row := range rows {
			if err := s.dst.UpsertWifi(row); err != nil {
				log.Printf("[WARN] upsert wifi: %v", err)
				continue
			}
			total++
		}
		if len(rows) < refreshPageSize {
			break
		}
		offset += len(rows)
	}
	return total, nil
}

func (s *RefreshService) refreshIPs() (int, error) {
	total, offset := 0, 0
	for {
		rows, err := s.src.FetchIPsForRefresh(offset, refreshPageSize)
		if err != nil {
			return total, err
		}
		for _, row := range rows {
			if err := s.dst.UpsertIP(row); err != nil {
				log.Printf("[WARN] upsert ip: %v", err)
				continue
			}
			total++
		}
		if len(rows) < refreshPageSize {
			break
		}
		offset += len(rows)
	}
	return total, nil
}
