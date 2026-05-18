package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/lenchik-en/lbs_server/internal/domain/model"
	"github.com/lenchik-en/lbs_server/internal/domain/port"
)

const sessionTTL = 8 * time.Hour

type session struct {
	expiresAt time.Time
}

// AdminService управляет сессиями админа и динамическими настройками.
type AdminService struct {
	login        string
	passwordHash string

	mu       sync.Mutex
	sessions map[string]session

	settingsMu   sync.RWMutex
	settings     model.Settings
	settingsRepo port.SettingsRepository
}

func NewAdminService(login, passwordHash string, defaults model.Settings, repo port.SettingsRepository) *AdminService {
	svc := &AdminService{
		login:        login,
		passwordHash: passwordHash,
		sessions:     make(map[string]session),
		settings:     defaults,
		settingsRepo: repo,
	}
	if saved, err := repo.Load(context.Background()); err != nil {
		log.Printf("[WARN] load settings from DB: %v", err)
	} else if saved != nil {
		svc.settings = *saved
	}
	go svc.cleanupSessions()
	return svc
}

// Login проверяет логин/пароль и возвращает токен сессии.
func (s *AdminService) Login(login, password string) (string, error) {
	if login != s.login {
		return "", fmt.Errorf("invalid credentials")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(s.passwordHash), []byte(password)); err != nil {
		return "", fmt.Errorf("invalid credentials")
	}

	token, err := generateToken()
	if err != nil {
		return "", err
	}

	s.mu.Lock()
	s.sessions[token] = session{expiresAt: time.Now().Add(sessionTTL)}
	s.mu.Unlock()

	return token, nil
}

// ValidateSession проверяет токен и возвращает true если сессия активна.
func (s *AdminService) ValidateSession(token string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.sessions[token]
	if !ok || time.Now().After(sess.expiresAt) {
		delete(s.sessions, token)
		return false
	}
	return true
}

// Logout удаляет сессию.
func (s *AdminService) Logout(token string) {
	s.mu.Lock()
	delete(s.sessions, token)
	s.mu.Unlock()
}

// GetSettings возвращает текущие настройки.
func (s *AdminService) GetSettings() model.Settings {
	s.settingsMu.RLock()
	defer s.settingsMu.RUnlock()
	return s.settings
}

// UpdateSettings обновляет настройки и сохраняет в БД.
func (s *AdminService) UpdateSettings(settings model.Settings) {
	s.settingsMu.Lock()
	s.settings = settings
	s.settingsMu.Unlock()
	if err := s.settingsRepo.Save(context.Background(), settings); err != nil {
		log.Printf("[WARN] persist settings: %v", err)
	}
}

func (s *AdminService) cleanupSessions() {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		s.mu.Lock()
		for token, sess := range s.sessions {
			if now.After(sess.expiresAt) {
				delete(s.sessions, token)
			}
		}
		s.mu.Unlock()
	}
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("rand: %w", err)
	}
	return hex.EncodeToString(b), nil
}
