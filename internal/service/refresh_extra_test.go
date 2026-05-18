package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/lenchik-en/lbs_server/internal/domain/model"
)

// channelDest реализует port.RefreshDest и сигнализирует через канал при первом UpsertCell.
type channelDest struct {
	ch chan struct{}
}

func (d *channelDest) UpsertCell(_ model.CellRefreshRow) error {
	select {
	case d.ch <- struct{}{}:
	default:
	}
	return nil
}
func (d *channelDest) UpsertWifi(_ model.WifiRefreshRow) error { return nil }
func (d *channelDest) UpsertIP(_ model.IPRefreshRow) error     { return nil }

// --- refreshWifis / refreshIPs: пагинация (offset += len(rows)) ---

func TestRefresh_MultiPageWifis(t *testing.T) {
	total := refreshPageSize + 150
	wifis := make([]model.WifiRefreshRow, total)
	svc := NewRefreshService(
		&mockRefreshSource{wifis: wifis},
		&mockRefreshDest{},
	)
	stats, err := svc.Refresh(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.WifisUpserted != total {
		t.Errorf("expected %d wifis, got %d", total, stats.WifisUpserted)
	}
}

func TestRefresh_MultiPageIPs(t *testing.T) {
	total := refreshPageSize + 200
	ips := make([]model.IPRefreshRow, total)
	svc := NewRefreshService(
		&mockRefreshSource{ips: ips},
		&mockRefreshDest{},
	)
	stats, err := svc.Refresh(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.IPsUpserted != total {
		t.Errorf("expected %d IPs, got %d", total, stats.IPsUpserted)
	}
}

// --- StartAutoRefresh ---

func TestStartAutoRefresh_ZeroPeriod_DoesNotRefresh(t *testing.T) {
	ch := make(chan struct{}, 1)
	svc := NewRefreshService(
		&mockRefreshSource{cells: makeCells(1)},
		&channelDest{ch: ch},
	)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	svc.StartAutoRefresh(ctx, func() int { return 0 })

	// period=0 — refresh не должен запускаться.
	select {
	case <-ch:
		t.Error("refresh must not trigger when period=0")
	case <-time.After(100 * time.Millisecond):
		// OK
	}
}

func TestStartAutoRefresh_NonZeroPeriod_TriggersRefresh(t *testing.T) {
	ch := make(chan struct{}, 1)
	svc := NewRefreshService(
		&mockRefreshSource{cells: makeCells(1)},
		&channelDest{ch: ch},
	)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	svc.StartAutoRefresh(ctx, func() int { return 20 }) // 20 мс

	select {
	case <-ch:
		// Refresh сработал — тест пройден.
	case <-time.After(2 * time.Second):
		t.Error("auto-refresh did not trigger within 2 seconds")
	}
}

func TestStartAutoRefresh_RefreshError_IsLogged(t *testing.T) {
	src := &mockRefreshSource{cellErr: errors.New("forced error")}
	svc := NewRefreshService(src, &mockRefreshDest{})

	ctx, cancel := context.WithCancel(context.Background())
	svc.StartAutoRefresh(ctx, func() int { return 10 })

	time.Sleep(80 * time.Millisecond)
	cancel()
	time.Sleep(20 * time.Millisecond)
}

func TestStartAutoRefresh_CancelStopsGoroutine(t *testing.T) {
	ch := make(chan struct{}, 10)
	svc := NewRefreshService(
		&mockRefreshSource{cells: makeCells(1)},
		&channelDest{ch: ch},
	)
	ctx, cancel := context.WithCancel(context.Background())

	svc.StartAutoRefresh(ctx, func() int { return 20 })

	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Fatal("auto-refresh did not trigger before cancel")
	}

	cancel()
	time.Sleep(100 * time.Millisecond)

	for len(ch) > 0 {
		<-ch
	}
	time.Sleep(50 * time.Millisecond)
	if len(ch) != 0 {
		t.Error("expected no more refreshes after context cancellation")
	}
}

// TestStartAutoRefresh_PeriodChange проверяет, что смена периода через getPeriod()
// применяется без перезапуска сервера.
func TestStartAutoRefresh_PeriodChange(t *testing.T) {
	ch := make(chan struct{}, 10)
	svc := NewRefreshService(
		&mockRefreshSource{cells: makeCells(1)},
		&channelDest{ch: ch},
	)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	period := 0 // начинаем с 0 — refresh выключен
	svc.StartAutoRefresh(ctx, func() int { return period })

	// При period=0 — refresh не срабатывает.
	select {
	case <-ch:
		t.Fatal("refresh must not fire when period=0")
	case <-time.After(60 * time.Millisecond):
	}

	// Включаем: меняем period → горутина подхватит при следующем poll-тике.
	// Чтобы не ждать 5 секунд, используем маленький период и даём время.
	period = 20
	select {
	case <-ch:
		// refresh включился после смены периода
	case <-time.After(6 * time.Second):
		t.Error("auto-refresh did not start after period change")
	}
}
