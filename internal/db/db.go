package db

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

type DataBase interface {
	Connect() error
}

type LocateDB struct {
	Dsn string
	DB  *sql.DB
}

// TODO: добавить подключение к бд в конструктор?
func NewLocateDB(dsn string) *LocateDB {
	return &LocateDB{Dsn: dsn}
}

func (l *LocateDB) Connect() error {
	dsn := l.Dsn
	if dsn == "" {
		return fmt.Errorf("Dsn path is empty")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("failed to connect to the DB: %v", err)
	}

	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping DB: %v", err)
	}

	l.DB = db
	fmt.Println("Connection OK")
	return nil
}
