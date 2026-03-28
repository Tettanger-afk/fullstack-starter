package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"planit/database"
	"planit/handlers"
)

func main() {
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "data"
	}
	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		dbName = "database.sqlite"
	}

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatal("Failed to create data directory:", err)
	}

	dbPath := filepath.Join(dataDir, dbName)
	db, err := database.New(dbPath)
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer db.Close()

	env := os.Getenv("APP_ENV")
	if env == "" {
		env = os.Getenv("NODE_ENV") // support legacy env var
	}
	if env != "production" && env != "test" {
		db.SeedDevData()
	}

	mux := handlers.NewMux(db)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	log.Printf("PlanIT listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
