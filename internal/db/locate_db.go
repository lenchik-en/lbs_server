package db

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

func OpenPostgres(dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to the database: %v", err)
	}

	//TODO: is it necessary?
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %v", err)
	}
	return db, nil
}

type LocateDB struct {
	DB *sql.DB
}

func NewLocateDB(dsn string) (*LocateDB, error) {
	db, err := OpenPostgres(dsn)
	if err != nil {
		return nil, fmt.Errorf("locateDB: %w", err)
	}
	fmt.Println("Connection to LocateDB is OK")
	return &LocateDB{
		DB: db,
	}, nil
}

func (l *LocateDB) GetConnection() *sql.DB { return l.DB }
