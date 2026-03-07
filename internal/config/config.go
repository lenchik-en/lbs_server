package config

import (
	"fmt"
	"os"
)

type Config struct {
	HTTPAddr string
	DBDSN    string
	EDBDSN   string
	UDBDSN   string
}

func Load() (*Config, error) {
	cfg := &Config{
		HTTPAddr: os.Getenv("HTTP_ADDR"),
		DBDSN:    os.Getenv("DB_DSN"),
		EDBDSN:   os.Getenv("EDB_DSN"),
		UDBDSN:   os.Getenv("UDB_DSN"),
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
