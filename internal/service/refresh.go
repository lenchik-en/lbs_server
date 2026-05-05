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

// StartAutoRefresh запускает фоновую горутину, которая вызывает Refresh с периодом,
// возвращаемым getPeriod(). Каждые 5 секунд проверяет, изменился ли период,
// и пересоздаёт тикер — это позволяет менять период через admin UI без перезапуска.
// period=0 означает "выключено"; горутина продолжает работать и включится при смене на >0.
func (s *RefreshService) StartAutoRefresh(ctx context.Context, getPeriod func() int) {
	go func() {
		var (
			ticker     *time.Ticker
			tickC      <-chan time.Time
			lastPeriod = -1
		)
		poll := time.NewTicker(5 * time.Second)
		defer func() {
			poll.Stop()
			if ticker != nil {
				ticker.Stop()
			}
		}()

		checkPeriod := func() {
			p := getPeriod()
			if p == lastPeriod {
				return
			}
			if ticker != nil {
				ticker.Stop()
				ticker = nil
				tickC = nil
			}
			lastPeriod = p
			if p > 0 {
				ticker = time.NewTicker(time.Duration(p) * time.Millisecond)
				tickC = ticker.C
				log.Printf("[INFO] auto-refresh period set to %dms", p)
			} else {
				log.Println("[INFO] auto-refresh disabled")
			}
		}

		checkPeriod()

		for {
			select {
			case <-tickC:
				if _, err := s.Refresh(ctx); err != nil {
					log.Printf("[WARN] auto-refresh failed: %v", err)
				}
			case <-poll.C:
				checkPeriod()
			case <-ctx.Done():
				return
			}
		}
	}()
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
