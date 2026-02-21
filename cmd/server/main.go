// cmd/server/main.go
package main

import (
	"diploma-back/internal/config"
	"diploma-back/internal/database"
	"diploma-back/internal/handlers"
	"diploma-back/internal/storage"
	"diploma-back/pkg/imaging"
	"log"
)

func main() {
	cfg := config.Inst()
	// Initialize database
	db, err := database.InitDB(&cfg.DB)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Auto-migrate models
	if err := database.MigrateDB(db); err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	minioClient, err := storage.NewMinIOClient(&cfg.MinIO)
	if err != nil {
		log.Fatal("Failed to initialize MinIO client:", err)
	}

	imgng := imaging.NewImaging(cfg.MODEL_URL)

	handlers := handlers.NewHandler(cfg, db, minioClient, imgng)
	handlers.InitRoutes()

	log.Printf("Server starting on port %s", cfg.App.Port)
	if err := handlers.Start(); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
