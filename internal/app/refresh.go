package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/lenchik-en/lbs_server/internal/db"
)

const refreshBatchSize = 1000

// RefreshStats содержит счётчики результатов одного прохода refresh.
type RefreshStats struct {
	CellsUpserted int `json:"cells_upserted"`
	WifisUpserted int `json:"wifis_upserted"`
	IPsUpserted   int `json:"ips_upserted"`
}

// Refresh переносит данные с координатами из UpdateDB в LocateDB.
func (a *App) Refresh(_ context.Context) (RefreshStats, error) {
	var stats RefreshStats
	var err error

	stats.CellsUpserted, err = refreshCells(a.refreshSrc, a.refreshDest)
	if err != nil {
		return stats, fmt.Errorf("refresh cells: %w", err)
	}

	stats.WifisUpserted, err = refreshWifis(a.refreshSrc, a.refreshDest)
	if err != nil {
		return stats, fmt.Errorf("refresh wifis: %w", err)
	}

	stats.IPsUpserted, err = refreshIPs(a.refreshSrc, a.refreshDest)
	if err != nil {
		return stats, fmt.Errorf("refresh ips: %w", err)
	}

	log.Printf("[INFO] refresh done: cells=%d wifis=%d ips=%d",
		stats.CellsUpserted, stats.WifisUpserted, stats.IPsUpserted)
	return stats, nil
}

func refreshCells(src db.RefreshSource, dst db.RefreshDest) (int, error) {
	total, offset := 0, 0
	for {
		rows, err := src.FetchCellsForRefresh(offset, refreshBatchSize)
		if err != nil {
			return total, err
		}
		for _, row := range rows {
			if err := dst.UpsertCell(row); err != nil {
				log.Printf("[WARN] upsert cell failed: %v", err)
				continue
			}
			total++
		}
		if len(rows) < refreshBatchSize {
			break
		}
		offset += len(rows)
	}
	return total, nil
}

func refreshWifis(src db.RefreshSource, dst db.RefreshDest) (int, error) {
	total, offset := 0, 0
	for {
		rows, err := src.FetchWifisForRefresh(offset, refreshBatchSize)
		if err != nil {
			return total, err
		}
		for _, row := range rows {
			if err := dst.UpsertWifi(row); err != nil {
				log.Printf("[WARN] upsert wifi failed: %v", err)
				continue
			}
			total++
		}
		if len(rows) < refreshBatchSize {
			break
		}
		offset += len(rows)
	}
	return total, nil
}

func refreshIPs(src db.RefreshSource, dst db.RefreshDest) (int, error) {
	total, offset := 0, 0
	for {
		rows, err := src.FetchIPsForRefresh(offset, refreshBatchSize)
		if err != nil {
			return total, err
		}
		for _, row := range rows {
			if err := dst.UpsertIP(row); err != nil {
				log.Printf("[WARN] upsert ip failed: %v", err)
				continue
			}
			total++
		}
		if len(rows) < refreshBatchSize {
			break
		}
		offset += len(rows)
	}
	return total, nil
}

func (a *App) startRefreshLoop(ctx context.Context) {
	period := time.Duration(a.cfg.RefreshPeriodMs) * time.Millisecond
	if period == 0 {
		log.Println("[INFO] auto-refresh disabled (REFRESH_PERIOD_MS=0)")
		return
	}
	go func() {
		ticker := time.NewTicker(period)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if _, err := a.Refresh(context.Background()); err != nil {
					log.Printf("[WARN] auto-refresh failed: %v", err)
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	log.Printf("[INFO] auto-refresh started (period=%v)", period)
}

// HandleRefresh POST /refresh — ручной запуск refresh.
func (a *App) HandleRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	stats, err := a.Refresh(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("refresh failed: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		log.Printf("[WARN] failed to encode refresh stats: %v", err)
	}
}
