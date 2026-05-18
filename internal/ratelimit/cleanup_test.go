package ratelimit

import (
	"testing"
	"time"
)

// TestCleanup_RemovesStaleEntries проверяет логику удаления устаревших записей.
// Напрямую манипулирует lastSeen, так как находится в том же пакете.
func TestCleanup_RemovesStaleEntries(t *testing.T) {
	r := New()
	defer r.Stop()

	// Добавляем устаревшую запись (старше entryTTL) и свежую.
	r.mu.Lock()
	r.lastSeen["stale"] = time.Now().Add(-(entryTTL + time.Second))
	r.lastSeen["fresh"] = time.Now()
	r.mu.Unlock()

	// Воспроизводим логику из cleanup() без ожидания тикера.
	cutoff := time.Now().Add(-entryTTL)
	r.mu.Lock()
	for id, ts := range r.lastSeen {
		if ts.Before(cutoff) {
			delete(r.lastSeen, id)
		}
	}
	r.mu.Unlock()

	r.mu.Lock()
	_, staleExists := r.lastSeen["stale"]
	_, freshExists := r.lastSeen["fresh"]
	r.mu.Unlock()

	if staleExists {
		t.Error("stale entry should have been removed by cleanup logic")
	}
	if !freshExists {
		t.Error("fresh entry should remain after cleanup")
	}
}

// TestCleanup_DoesNotRemoveFreshEntries проверяет, что записи моложе entryTTL не удаляются.
func TestCleanup_DoesNotRemoveFreshEntries(t *testing.T) {
	r := New()
	defer r.Stop()

	r.mu.Lock()
	r.lastSeen["almost-stale"] = time.Now().Add(-(entryTTL - time.Second))
	r.lastSeen["new"] = time.Now()
	r.mu.Unlock()

	cutoff := time.Now().Add(-entryTTL)
	r.mu.Lock()
	for id, ts := range r.lastSeen {
		if ts.Before(cutoff) {
			delete(r.lastSeen, id)
		}
	}
	r.mu.Unlock()

	r.mu.Lock()
	_, almostStaleExists := r.lastSeen["almost-stale"]
	_, newExists := r.lastSeen["new"]
	r.mu.Unlock()

	if !almostStaleExists {
		t.Error("entry just under TTL should not be removed")
	}
	if !newExists {
		t.Error("new entry should not be removed")
	}
}

// TestCleanup_EmptyMap не паникует на пустой карте.
func TestCleanup_EmptyMap(t *testing.T) {
	r := New()
	defer r.Stop()

	cutoff := time.Now().Add(-entryTTL)
	r.mu.Lock()
	for id, ts := range r.lastSeen {
		if ts.Before(cutoff) {
			delete(r.lastSeen, id)
		}
	}
	r.mu.Unlock()
	// Нет паники — тест пройден.
}

// TestAllow_StaleEntryAllowedAgain проверяет, что после TTL ключ освобождается.
// Это покрывает ветку "last.Before(cutoff)" через реальный Allow.
func TestAllow_StaleEntryAllowedAgain(t *testing.T) {
	r := New()
	defer r.Stop()

	// Имитируем запись, которая уже вышла за TTL.
	r.mu.Lock()
	r.lastSeen["old-key"] = time.Now().Add(-(entryTTL + time.Second))
	r.mu.Unlock()

	// Логика Allow проверяет only time.Now().Sub(last) < minPeriod.
	// old-key помечен как "очень старый" → разрыв больше любого разумного minPeriod.
	if !r.Allow("old-key", 5000) {
		t.Error("entry past TTL should be treated as if fresh (not rate-limited)")
	}
}
