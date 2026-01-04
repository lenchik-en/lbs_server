package main

import (
	"github.com/lenchik-en/lbs_server/internal/app"
)

func main() {
	//err := godotenv.Load(".env")
	//if err != nil {
	//	log.Fatalf("failed to load .env file")
	//}

	app.Run()
}
