package main

import (
	"context"
	"log"
	"net/http"

	"vinylhound/internal/store"
)

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatal(err)
	}

	db, err := openDatabase(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	dataStore := store.New(db)

	if err := bootstrapDemoData(context.Background(), db, dataStore); err != nil {
		log.Fatal(err)
	}

	handler := newHTTPHandler(cfg, dataStore)

	log.Printf("API available at http://localhost%v", cfg.Addr)
	if err := http.ListenAndServe(cfg.Addr, handler); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
