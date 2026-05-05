package service

import (
	"context"
	"errors"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/lenchik-en/lbs_server/internal/domain/model"
)

// mockSettingsRepo implements port.SettingsRepository.
type mockSettingsRepo struct {
	loaded  *model.Settings
	loadErr error
	saved   *model.Settings
	saveErr error
}

func (r *mockSettingsRepo) Load(_ context.Context) (*model.Settings, error) {
	return r.loaded, r.loadErr
}

func (r *mockSettingsRepo) Save(_ context.Context, s model.Settings) error {
	r.saved = &s
	return r.saveErr
}

func makeAdminSvc(t *testing.T, login, pass string, loaded *model.Settings) (*AdminService, *mockSettingsRepo) {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("bcrypt: %v", err)
	}
	repo := &mockSettingsRepo{loaded: loaded}
	defaults := model.Settings{RequireKeyLocate: false, RequireKeyUpdate: false}
	svc := NewAdminService(login, string(hash), defaults, repo)
	return svc, repo
}

// --- Login ---

func TestLogin_WrongLogin(t *testing.T) {
	svc, _ := makeAdminSvc(t, "admin", "pass123", nil)
	_, err := svc.Login("wrong", "pass123")
	if err == nil {
		t.Fatal("expected error for wrong login")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	svc, _ := makeAdminSvc(t, "admin", "pass123", nil)
	_, err := svc.Login("admin", "wrongpass")
	if err == nil {
		t.Fatal("expected error for wrong password")
	}
}

func TestLogin_Success(t *testing.T) {
	svc, _ := makeAdminSvc(t, "admin", "pass123", nil)
	token, err := svc.Login("admin", "pass123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(token) != 64 {
		t.Errorf("expected 64-char hex token, got len=%d: %q", len(token), token)
	}
}

func TestLogin_EachCallReturnsDifferentToken(t *testing.T) {
	svc, _ := makeAdminSvc(t, "admin", "pass", nil)
	t1, _ := svc.Login("admin", "pass")
	t2, _ := svc.Login("admin", "pass")
	if t1 == t2 {
		t.Error("expected different tokens for consecutive logins")
	}
}

// --- ValidateSession ---

func TestValidateSession_ValidToken(t *testing.T) {
	svc, _ := makeAdminSvc(t, "admin", "pass", nil)
	token, _ := svc.Login("admin", "pass")
	if !svc.ValidateSession(token) {
		t.Error("expected valid session to be recognized")
	}
}

func TestValidateSession_UnknownToken(t *testing.T) {
	svc, _ := makeAdminSvc(t, "admin", "pass", nil)
	if svc.ValidateSession("does-not-exist") {
		t.Error("expected unknown token to be invalid")
	}
}

func TestValidateSession_EmptyToken(t *testing.T) {
	svc, _ := makeAdminSvc(t, "admin", "pass", nil)
	if svc.ValidateSession("") {
		t.Error("expected empty token to be invalid")
	}
}

// --- Logout ---

func TestLogout_InvalidatesSession(t *testing.T) {
	svc, _ := makeAdminSvc(t, "admin", "pass", nil)
	token, _ := svc.Login("admin", "pass")
	svc.Logout(token)
	if svc.ValidateSession(token) {
		t.Error("expected session to be invalid after logout")
	}
}

func TestLogout_UnknownTokenNoPanic(t *testing.T) {
	svc, _ := makeAdminSvc(t, "admin", "pass", nil)
	svc.Logout("non-existent-token") // must not panic
}

func TestLogout_DoesNotAffectOtherSessions(t *testing.T) {
	svc, _ := makeAdminSvc(t, "admin", "pass", nil)
	t1, _ := svc.Login("admin", "pass")
	t2, _ := svc.Login("admin", "pass")
	svc.Logout(t1)
	if svc.ValidateSession(t1) {
		t.Error("t1 should be invalid after logout")
	}
	if !svc.ValidateSession(t2) {
		t.Error("t2 should remain valid")
	}
}

// --- GetSettings ---

func TestGetSettings_DefaultsWhenRepoReturnsNil(t *testing.T) {
	defaults := model.Settings{RequireKeyLocate: true, RefreshPeriodMs: 5000}
	hash, _ := bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.MinCost)
	repo := &mockSettingsRepo{loaded: nil}
	svc := NewAdminService("admin", string(hash), defaults, repo)
	got := svc.GetSettings()
	if !got.RequireKeyLocate || got.RefreshPeriodMs != 5000 {
		t.Errorf("expected defaults, got %+v", got)
	}
}

func TestGetSettings_LoadedFromRepo(t *testing.T) {
	saved := &model.Settings{RequireKeyLocate: true, RefreshPeriodMs: 9999}
	hash, _ := bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.MinCost)
	repo := &mockSettingsRepo{loaded: saved}
	svc := NewAdminService("admin", string(hash), model.Settings{}, repo)
	got := svc.GetSettings()
	if !got.RequireKeyLocate || got.RefreshPeriodMs != 9999 {
		t.Errorf("expected loaded settings, got %+v", got)
	}
}

func TestGetSettings_LoadErrorUsesDefaults(t *testing.T) {
	defaults := model.Settings{RefreshPeriodMs: 7777}
	hash, _ := bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.MinCost)
	repo := &mockSettingsRepo{loadErr: errors.New("db error")}
	svc := NewAdminService("admin", string(hash), defaults, repo)
	got := svc.GetSettings()
	if got.RefreshPeriodMs != 7777 {
		t.Errorf("expected defaults on load error, got %+v", got)
	}
}

// --- UpdateSettings ---

func TestUpdateSettings_UpdatesInMemory(t *testing.T) {
	svc, _ := makeAdminSvc(t, "admin", "pass", nil)
	newSettings := model.Settings{RequireKeyLocate: true, RefreshPeriodMs: 1234}
	svc.UpdateSettings(newSettings)
	got := svc.GetSettings()
	if !got.RequireKeyLocate || got.RefreshPeriodMs != 1234 {
		t.Errorf("expected updated settings, got %+v", got)
	}
}

func TestUpdateSettings_PersistsToRepo(t *testing.T) {
	svc, repo := makeAdminSvc(t, "admin", "pass", nil)
	newSettings := model.Settings{RequireKeyUpdate: true, LocateMinPeriodKey: 500}
	svc.UpdateSettings(newSettings)
	if repo.saved == nil {
		t.Fatal("expected settings to be persisted to repo")
	}
	if !repo.saved.RequireKeyUpdate || repo.saved.LocateMinPeriodKey != 500 {
		t.Errorf("persisted wrong settings: %+v", repo.saved)
	}
}

func TestUpdateSettings_SaveErrorDoesNotRevertMemory(t *testing.T) {
	svc, repo := makeAdminSvc(t, "admin", "pass", nil)
	repo.saveErr = errors.New("db error")
	svc.UpdateSettings(model.Settings{RequireKeyLocate: true})
	if !svc.GetSettings().RequireKeyLocate {
		t.Error("in-memory settings must update even when persist fails")
	}
}
