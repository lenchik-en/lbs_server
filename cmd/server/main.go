package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/lenchik-en/lbs_server/internal/app"
	"github.com/lenchik-en/lbs_server/internal/db"
)

func main() {
	godotenv.Load(".env")

	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		log.Fatalf("no path in DB_DSN")
	}

	locatedb := db.NewLocateDB(dsn)
	err := locatedb.Connect()
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	defer locatedb.DB.Close()

	app.Run(locatedb)
}
