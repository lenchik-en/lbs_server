package app

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/lenchik-en/lbs_server/internal/config"
	"github.com/lenchik-en/lbs_server/internal/db"
)

type App struct {
	locateDB   db.CellFinder
	externalDB db.CellFinder
	updateDB   *db.UpdateDB
	session    *db.Session
}

func (a *App) New(locate db.CellFinder, external db.CellFinder, update *db.UpdateDB) {
	a.locateDB = locate
	a.externalDB = external
	a.updateDB = update
	a.session = db.NewLogger(locate.GetConnection())
}

func (a *App) Run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %v", err)
	}

	locateDB, err := db.NewLocateDB(cfg.DBDSN)
	if err != nil {
		return fmt.Errorf("failed to connect locateDB: %v", err)
	}
	defer locateDB.DB.Close()

	externalDB, err := db.NewExternalDB(cfg.EDBDSN)
	if err != nil {
		return fmt.Errorf("failed to connect externalDB: %v", err)
	}
	defer externalDB.DB.Close()

	updateDB, err := db.NewUpdateDB(cfg.UDBDSN)
	if err != nil {
		return fmt.Errorf("failed to connect updateDB: %v", err)
	}
	defer updateDB.DB.Close()

	a.New(locateDB, externalDB, updateDB)

	return nil
}

func (a *App) CloseDBConnections() {
	if a.locateDB != nil {
		a.locateDB.GetConnection().Close()
	}
	if a.externalDB != nil {
		a.externalDB.GetConnection().Close()
	}
	if a.updateDB != nil {
		a.updateDB.DB.Close()
	}
}

func (a *App) HandleHealth(w http.ResponseWriter, r *http.Request) {
	resp := map[string]string{"status": "ok"}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
