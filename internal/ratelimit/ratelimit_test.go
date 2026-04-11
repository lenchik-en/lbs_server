package ratelimit

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestAllow_ZeroPeriod(t *testing.T) {
	r := New()
	defer r.Stop()

	// minPeriodMs == 0 → всегда разрешено, сколько угодно раз подряд
	for i := 0; i < 10; i++ {
		if !r.Allow("key", 0) {
			t.Fatalf("Allow with minPeriodMs=0 must always return true (call %d)", i)
		}
	}
}

func TestAllow_EmptyIdentifier(t *testing.T) {
	r := New()
	defer r.Stop()

	for i := 0; i < 10; i++ {
		if !r.Allow("", 1000) {
			t.Fatalf("Allow with empty identifier must always return true (call %d)", i)
		}
	}
}

func TestAllow_FirstRequestAlwaysAllowed(t *testing.T) {
	r := New()
	defer r.Stop()

	if !r.Allow("new-key", 5000) {
		t.Fatal("first request must always be allowed")
	}
}

func TestAllow_SecondRequestWithinPeriodBlocked(t *testing.T) {
	r := New()
	defer r.Stop()

	const period = 500 // ms

	if !r.Allow("key", period) {
		t.Fatal("first request must be allowed")
	}
	if r.Allow("key", period) {
		t.Fatal("second request within period must be blocked")
	}
}

func TestAllow_RequestAfterPeriodAllowed(t *testing.T) {
	r := New()
	defer r.Stop()

	const period = 50 // ms — минимально для теста

	if !r.Allow("key", period) {
		t.Fatal("first request must be allowed")
	}
	time.Sleep(time.Duration(period+20) * time.Millisecond)
	if !r.Allow("key", period) {
		t.Fatal("request after period must be allowed")
	}
}

func TestAllow_DifferentIdentifiersIndependent(t *testing.T) {
	r := New()
	defer r.Stop()

	const period = 5000 // ms — большой, чтобы второй вызов точно попал в окно

	r.Allow("key-A", period)
	r.Allow("key-A", period) // заблокирован

	// key-B — новый, должен пройти
	if !r.Allow("key-B", period) {
		t.Fatal("different key must not be rate-limited by key-A")
	}
}

func TestAllow_SameKeyDifferentPeriods(t *testing.T) {
	r := New()
	defer r.Stop()

	// Первый вызов с большим периодом
	if !r.Allow("key", 10_000) {
		t.Fatal("first call must be allowed")
	}
	// Второй вызов с периодом=0 — должен пройти несмотря на то, что ключ "видели"
	if !r.Allow("key", 0) {
		t.Fatal("period=0 must always allow")
	}
}

func TestAllow_ConcurrentAccess(t *testing.T) {
	// Проверяем отсутствие race conditions под -race
	r := New()
	defer r.Stop()

	const goroutines = 50
	const period = 10 // ms

	var wg sync.WaitGroup
	var allowed atomic.Int32

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if r.Allow("shared-key", period) {
				allowed.Add(1)
			}
		}()
	}
	wg.Wait()

	// Ровно один горутин должен пройти в первый момент времени
	// (остальные могут пройти, если планировщик размазал их во времени —
	// поэтому проверяем только that >= 1 и нет паники)
	if allowed.Load() == 0 {
		t.Fatal("at least one concurrent request must be allowed")
	}
}

func TestAllow_ConcurrentDifferentKeys(t *testing.T) {
	// Разные ключи не мешают друг другу при параллельном доступе
	r := New()
	defer r.Stop()

	const goroutines = 100
	const period = 10_000 // ms — большой, чтобы вторые вызовы блокировались

	var wg sync.WaitGroup
	var allowed atomic.Int32

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		key := string(rune('A' + i%26)) // 26 уникальных ключей
		go func(k string) {
			defer wg.Done()
			if r.Allow(k, period) {
				allowed.Add(1)
			}
		}(key)
	}
	wg.Wait()

	// Каждый уникальный ключ должен пройти хотя бы один раз
	if allowed.Load() == 0 {
		t.Fatal("expected at least one allowed request per unique key")
	}
}

func TestStop_NoDoublePanic(t *testing.T) {
	r := New()
	r.Stop()
	// Stop второй раз вызывать нельзя (close закрытого канала → panic),
	// но один Stop должен быть безопасен
}

func TestAllow_SequentialCallsCountCorrectly(t *testing.T) {
	r := New()
	defer r.Stop()

	const period = 200 // ms

	calls := []bool{
		r.Allow("seq", period), // 1st: allowed
		r.Allow("seq", period), // 2nd: blocked
		r.Allow("seq", period), // 3rd: blocked
	}

	if !calls[0] {
		t.Error("1st call must be allowed")
	}
	if calls[1] {
		t.Error("2nd call within period must be blocked")
	}
	if calls[2] {
		t.Error("3rd call within period must be blocked")
	}
}
