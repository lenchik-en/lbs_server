package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	HTTPAddr string
	DBDSN    string
	EDBDSN   string
	UDBDSN   string

	// Rate limiting: minimum period in milliseconds between requests per identifier.
	// 0 = disabled. Configured independently for /locate and /update,
	// and for three identifier scopes: API key, session UUID, and (key, UUID) pair.
	LocateMinPeriodKey     int
	LocateMinPeriodUUID    int
	LocateMinPeriodKeyUUID int
	UpdateMinPeriodKey     int
	UpdateMinPeriodUUID    int
	UpdateMinPeriodKeyUUID int

	// Refresh: период автоматической перекачки UpdateDB → LocateDB, мс.
	// 0 = авто-refresh отключён, только ручной через POST /refresh.
	RefreshPeriodMs int
}

func Load() (*Config, error) {
	cfg := &Config{
		HTTPAddr:               os.Getenv("HTTP_ADDR"),
		DBDSN:                  os.Getenv("DB_DSN"),
		EDBDSN:                 os.Getenv("EDB_DSN"),
		UDBDSN:                 os.Getenv("UDB_DSN"),
		LocateMinPeriodKey:     envInt("LOCATE_MIN_PERIOD_KEY"),
		LocateMinPeriodUUID:    envInt("LOCATE_MIN_PERIOD_UUID"),
		LocateMinPeriodKeyUUID: envInt("LOCATE_MIN_PERIOD_KEY_UUID"),
		UpdateMinPeriodKey:     envInt("UPDATE_MIN_PERIOD_KEY"),
		UpdateMinPeriodUUID:    envInt("UPDATE_MIN_PERIOD_UUID"),
		UpdateMinPeriodKeyUUID: envInt("UPDATE_MIN_PERIOD_KEY_UUID"),
		RefreshPeriodMs:        envInt("REFRESH_PERIOD_MS"),
	}

	if cfg.HTTPAddr == "" {
		return nil, fmt.Errorf("no path in HTTP_ADDR")
	}

	if cfg.DBDSN == "" {
		return nil, fmt.Errorf("no path in DB_DSN")
	}

	if cfg.EDBDSN == "" {
		return nil, fmt.Errorf("no path in EDB_DSN")
	}

	if cfg.UDBDSN == "" {
		return nil, fmt.Errorf("no path in UDB_DSN")
	}

	return cfg, nil
}

// envInt reads an integer from an environment variable. Returns 0 if unset or invalid.
func envInt(key string) int {
	v, _ := strconv.Atoi(os.Getenv(key))
	return v
}
