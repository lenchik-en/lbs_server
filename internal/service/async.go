package service

import "sync"

// asyncGroup абстрагирует запуск фоновых горутин с ожиданием завершения.
// Позволяет тестировать сервис без реального sync.WaitGroup.
type asyncGroup interface {
	Go(fn func())
	Wait()
}

// WaitGroupAdapter оборачивает sync.WaitGroup в asyncGroup.
type WaitGroupAdapter struct {
	wg sync.WaitGroup
}

func (a *WaitGroupAdapter) Go(fn func()) {
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		fn()
	}()
}

func (a *WaitGroupAdapter) Wait() {
	a.wg.Wait()
}
