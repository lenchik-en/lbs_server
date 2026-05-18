package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/lenchik-en/lbs_server/internal/domain/model"
)

// mockAPIKeyRepo implements port.APIKeyRepository with call tracking.
type mockAPIKeyRepo struct {
	keys        map[string]*model.APIKey
	getCalls    int
	getErr      error
	deactCalled bool
	deactErr    error
	createFn    func(model.KeyScope) (*model.APIKey, error)
	listFn      func() ([]*model.APIKey, error)
}

func (r *mockAPIKeyRepo) GetByKey(_ context.Context, key string) (*model.APIKey, error) {
	r.getCalls++
	if r.getErr != nil {
		return nil, r.getErr
	}
	if r.keys == nil {
		return nil, nil
	}
	return r.keys[key], nil
}

func (r *mockAPIKeyRepo) Create(_ context.Context, scope model.KeyScope) (*model.APIKey, error) {
	if r.createFn != nil {
		return r.createFn(scope)
	}
	return &model.APIKey{Key: "new_key", Scope: scope, IsActive: true}, nil
}

func (r *mockAPIKeyRepo) Deactivate(_ context.Context, _ string) error {
	r.deactCalled = true
	return r.deactErr
}

func (r *mockAPIKeyRepo) ListAll(_ context.Context) ([]*model.APIKey, error) {
	if r.listFn != nil {
		return r.listFn()
	}
	return nil, nil
}

// --- Validate ---

func TestAPIKeyValidate_NilKey(t *testing.T) {
	svc := NewAPIKeyService(&mockAPIKeyRepo{keys: map[string]*model.APIKey{}})
	ok, err := svc.Validate(context.Background(), "missing-key", model.ScopeLocate)
	if err != nil || ok {
		t.Errorf("expected false,nil for missing key; got ok=%v err=%v", ok, err)
	}
}

func TestAPIKeyValidate_InactiveKey(t *testing.T) {
	svc := NewAPIKeyService(&mockAPIKeyRepo{keys: map[string]*model.APIKey{
		"k1": {Key: "k1", Scope: model.ScopeLocate, IsActive: false},
	}})
	ok, err := svc.Validate(context.Background(), "k1", model.ScopeLocate)
	if err != nil || ok {
		t.Errorf("expected false,nil for inactive key; got ok=%v err=%v", ok, err)
	}
}

func TestAPIKeyValidate_MatchingScope(t *testing.T) {
	svc := NewAPIKeyService(&mockAPIKeyRepo{keys: map[string]*model.APIKey{
		"k1": {Key: "k1", Scope: model.ScopeLocate, IsActive: true},
	}})
	ok, err := svc.Validate(context.Background(), "k1", model.ScopeLocate)
	if err != nil || !ok {
		t.Errorf("expected true,nil; got ok=%v err=%v", ok, err)
	}
}

func TestAPIKeyValidate_ScopeMismatch(t *testing.T) {
	svc := NewAPIKeyService(&mockAPIKeyRepo{keys: map[string]*model.APIKey{
		"k1": {Key: "k1", Scope: model.ScopeLocate, IsActive: true},
	}})
	ok, err := svc.Validate(context.Background(), "k1", model.ScopeUpdate)
	if err != nil || ok {
		t.Errorf("expected false,nil for scope mismatch; got ok=%v err=%v", ok, err)
	}
}

func TestAPIKeyValidate_BothScopeGrantsAll(t *testing.T) {
	svc := NewAPIKeyService(&mockAPIKeyRepo{keys: map[string]*model.APIKey{
		"k1": {Key: "k1", Scope: model.ScopeBoth, IsActive: true},
	}})
	for _, scope := range []model.KeyScope{model.ScopeLocate, model.ScopeUpdate, model.ScopeBoth} {
		ok, err := svc.Validate(context.Background(), "k1", scope)
		if err != nil || !ok {
			t.Errorf("expected true,nil for scope=%s with 'both' key; got ok=%v err=%v", scope, ok, err)
		}
	}
}

func TestAPIKeyValidate_RepoError(t *testing.T) {
	svc := NewAPIKeyService(&mockAPIKeyRepo{getErr: errors.New("db error")})
	ok, err := svc.Validate(context.Background(), "k1", model.ScopeLocate)
	if err == nil || ok {
		t.Errorf("expected error propagated; got ok=%v err=%v", ok, err)
	}
}

// --- Cache ---

func TestAPIKeyValidate_CacheHitDoesNotCallRepo(t *testing.T) {
	repo := &mockAPIKeyRepo{keys: map[string]*model.APIKey{
		"k1": {Key: "k1", Scope: model.ScopeLocate, IsActive: true},
	}}
	svc := NewAPIKeyService(repo)
	svc.Validate(context.Background(), "k1", model.ScopeLocate)
	svc.Validate(context.Background(), "k1", model.ScopeLocate)
	if repo.getCalls != 1 {
		t.Errorf("expected 1 repo call due to cache, got %d", repo.getCalls)
	}
}

func TestAPIKeyValidate_CacheExpiry(t *testing.T) {
	repo := &mockAPIKeyRepo{keys: map[string]*model.APIKey{
		"k1": {Key: "k1", Scope: model.ScopeLocate, IsActive: true},
	}}
	svc := NewAPIKeyService(repo)
	svc.Validate(context.Background(), "k1", model.ScopeLocate)

	// Manually expire cache entry.
	svc.mu.Lock()
	if c, ok := svc.cache["k1"]; ok {
		svc.cache["k1"] = cachedKey{key: c.key, expiresAt: time.Now().Add(-time.Second)}
	}
	svc.mu.Unlock()

	svc.Validate(context.Background(), "k1", model.ScopeLocate)
	if repo.getCalls != 2 {
		t.Errorf("expected 2 repo calls after cache expiry, got %d", repo.getCalls)
	}
}

// --- Revoke ---

func TestAPIKeyRevoke_InvalidatesCache(t *testing.T) {
	repo := &mockAPIKeyRepo{keys: map[string]*model.APIKey{
		"k1": {Key: "k1", Scope: model.ScopeLocate, IsActive: true},
	}}
	svc := NewAPIKeyService(repo)
	svc.Validate(context.Background(), "k1", model.ScopeLocate) // populate cache
	svc.Revoke(context.Background(), "k1")
	svc.Validate(context.Background(), "k1", model.ScopeLocate) // must hit repo again
	if repo.getCalls != 2 {
		t.Errorf("expected 2 repo calls after revoke, got %d", repo.getCalls)
	}
}

func TestAPIKeyRevoke_CallsDeactivate(t *testing.T) {
	repo := &mockAPIKeyRepo{keys: map[string]*model.APIKey{}}
	svc := NewAPIKeyService(repo)
	svc.Revoke(context.Background(), "k1")
	if !repo.deactCalled {
		t.Error("expected Deactivate to be called")
	}
}

func TestAPIKeyRevoke_DeactivateError(t *testing.T) {
	repo := &mockAPIKeyRepo{
		keys:     map[string]*model.APIKey{},
		deactErr: errors.New("db error"),
	}
	svc := NewAPIKeyService(repo)
	if err := svc.Revoke(context.Background(), "k1"); err == nil {
		t.Error("expected deactivate error to be propagated")
	}
}

func TestAPIKeyRevoke_KeyBecomesInvalidAfterRevoke(t *testing.T) {
	key := &model.APIKey{Key: "k1", Scope: model.ScopeLocate, IsActive: true}
	repo := &mockAPIKeyRepo{keys: map[string]*model.APIKey{"k1": key}}
	svc := NewAPIKeyService(repo)

	ok, _ := svc.Validate(context.Background(), "k1", model.ScopeLocate)
	if !ok {
		t.Fatal("key should be valid before revoke")
	}

	key.IsActive = false // simulate repo update
	svc.Revoke(context.Background(), "k1")

	ok, err := svc.Validate(context.Background(), "k1", model.ScopeLocate)
	if err != nil || ok {
		t.Errorf("expected key to be invalid after revoke; got ok=%v err=%v", ok, err)
	}
}

// --- Generate / ListAll ---

func TestAPIKeyGenerate_CallsCreate(t *testing.T) {
	var gotScope model.KeyScope
	repo := &mockAPIKeyRepo{
		createFn: func(scope model.KeyScope) (*model.APIKey, error) {
			gotScope = scope
			return &model.APIKey{Key: "new", Scope: scope, IsActive: true}, nil
		},
	}
	svc := NewAPIKeyService(repo)
	key, err := svc.Generate(context.Background(), model.ScopeUpdate)
	if err != nil || key == nil {
		t.Fatalf("unexpected result: err=%v key=%v", err, key)
	}
	if gotScope != model.ScopeUpdate {
		t.Errorf("expected scope=%s, got %s", model.ScopeUpdate, gotScope)
	}
}

func TestAPIKeyListAll_ReturnsKeys(t *testing.T) {
	expected := []*model.APIKey{
		{Key: "k1", Scope: model.ScopeLocate, IsActive: true},
		{Key: "k2", Scope: model.ScopeBoth, IsActive: false},
	}
	repo := &mockAPIKeyRepo{
		listFn: func() ([]*model.APIKey, error) { return expected, nil },
	}
	svc := NewAPIKeyService(repo)
	keys, err := svc.ListAll(context.Background())
	if err != nil || len(keys) != 2 {
		t.Errorf("expected 2 keys err=nil; got len=%d err=%v", len(keys), err)
	}
}

func TestAPIKeyListAll_RepoError(t *testing.T) {
	repo := &mockAPIKeyRepo{
		listFn: func() ([]*model.APIKey, error) { return nil, errors.New("db error") },
	}
	svc := NewAPIKeyService(repo)
	_, err := svc.ListAll(context.Background())
	if err == nil {
		t.Error("expected error from repo")
	}
}
