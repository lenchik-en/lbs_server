package model

// Settings — динамические настройки сервера, изменяемые через admin UI.
type Settings struct {
	RequireKeyLocate    bool
	RequireKeyUpdate    bool
	RefreshPeriodMs     int
	LocateMinPeriodKey  int
	LocateMinPeriodAnon int
	UpdateMinPeriodKey  int
	UpdateMinPeriodAnon int
}
