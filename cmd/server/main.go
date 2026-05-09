package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"deadcomments/internal/app"
	"deadcomments/internal/db"
)

func main() {
	cfg, err := app.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}
	database, err := db.Open(cfg.DatabasePath)
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.Migrate(ctx, database); err != nil {
		log.Fatal(err)
	}
	if err := app.SeedDevelopment(ctx, database, cfg); err != nil {
		log.Fatal(err)
	}
	application, err := app.New(cfg, database)
	if err != nil {
		log.Fatal(err)
	}
	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           application.Router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	log.Printf("deadcomments listening on %s", cfg.BaseURL)
	log.Fatal(server.ListenAndServe())
}
