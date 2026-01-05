package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/lenchik-en/lbs_server/internal/api"
)

type UpdateStore interface {
	InsertCell(ctx context.Context, cell api.Cell) error
	//InsertWIFI
}

type UpdateDB struct {
	DB *sql.DB
}

func NewUpdateDB(dsn string) (*UpdateDB, error) {
	db, err := OpenPostgres(dsn)
	if err != nil {
		return nil, fmt.Errorf("locateDB: %w", err)
	}
	fmt.Println("Connection to UpdateDB is OK")
	return &UpdateDB{
		DB: db,
	}, nil
}

func (u *UpdateDB) InsertCell(ctx context.Context, cell api.Cell) error {
	return nil
}
