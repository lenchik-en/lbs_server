package service

import (
	"sync"
	"testing"
)

func TestWaitGroupAdapter_RunsFunction(t *testing.T) {
	wga := &WaitGroupAdapter{}
	called := false
	wga.Go(func() { called = true })
	wga.Wait()
	if !called {
		t.Error("expected function to be executed by Go()")
	}
}

func TestWaitGroupAdapter_RunsMultipleFunctions(t *testing.T) {
	wga := &WaitGroupAdapter{}
	const n = 10
	var mu sync.Mutex
	count := 0
	for i := 0; i < n; i++ {
		wga.Go(func() {
			mu.Lock()
			count++
			mu.Unlock()
		})
	}
	wga.Wait()
	if count != n {
		t.Errorf("expected %d functions executed, got %d", n, count)
	}
}

func TestWaitGroupAdapter_WaitBlocksUntilAllDone(t *testing.T) {
	wga := &WaitGroupAdapter{}
	// close не блокирует, поэтому горутина завершится до того как wga.Wait() вернётся.
	done := make(chan struct{})
	wga.Go(func() {
		close(done)
	})
	wga.Wait()
	// Если Wait вернулся, горутина уже закрыла канал.
	select {
	case <-done:
		// OK: канал закрыт, горутина выполнилась
	default:
		t.Error("goroutine should have completed before Wait returned")
	}
}

func TestWaitGroupAdapter_EmptyWait(t *testing.T) {
	wga := &WaitGroupAdapter{}
	// Wait без вызовов Go() не должен блокировать.
	wga.Wait()
}
