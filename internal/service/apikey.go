package service

import (
	"context"
	"sync"
	"time"

	"github.com/lenchik-en/lbs_server/internal/domain/model"
	"github.com/lenchik-en/lbs_server/internal/domain/port"
)

const keyCacheTTL = 30 * time.Second

type cachedKey struct {
	key       *model.APIKey
	expiresAt time.Time
}

type APIKeyService struct {
	repo  port.APIKeyRepository
	mu    sync.Mutex
	cache map[string]cachedKey
}

func NewAPIKeyService(repo port.APIKeyRepository) *APIKeyService {
	return &APIKeyService{
		repo:  repo,
		cache: make(map[string]cachedKey),
	}
}

// Validate проверяет, что ключ существует, активен и подходит по скоупу.
func (s *APIKeyService) Validate(ctx context.Context, key string, scope model.KeyScope) (bool, error) {
	k, err := s.getWithCache(ctx, key)
	if err != nil {
		return false, err
	}
	if k == nil || !k.IsActive {
		return false, nil
	}
	return k.Scope == scope || k.Scope == model.ScopeBoth, nil
}

func (s *APIKeyService) Generate(ctx context.Context, scope model.KeyScope) (*model.APIKey, error) {
	return s.repo.Create(ctx, scope)
}

func (s *APIKeyService) Revoke(ctx context.Context, key string) error {
	s.invalidateCache(key)
	return s.repo.Deactivate(ctx, key)
}

func (s *APIKeyService) ListAll(ctx context.Context) ([]*model.APIKey, error) {
	return s.repo.ListAll(ctx)
}

func (s *APIKeyService) getWithCache(ctx context.Context, key string) (*model.APIKey, error) {
	s.mu.Lock()
	if c, ok := s.cache[key]; ok && time.Now().Before(c.expiresAt) {
		s.mu.Unlock()
		return c.key, nil
	}
	s.mu.Unlock()

	k, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	s.cache[key] = cachedKey{key: k, expiresAt: time.Now().Add(keyCacheTTL)}
	s.mu.Unlock()

	return k, nil
}

func (s *APIKeyService) invalidateCache(key string) {
	s.mu.Lock()
	delete(s.cache, key)
	s.mu.Unlock()
}
