// cmd/server/main.go
package main

import (
	"diploma-back/internal/config"
	"diploma-back/internal/database"
	"diploma-back/internal/handlers"
	"diploma-back/internal/repository"
	"diploma-back/internal/services"
	"diploma-back/internal/storage"
	"diploma-back/pkg/imaging"
	"github.com/gin-gonic/gin"
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

	imgng := imaging.NewImaging(cfg.MODEL_URL, cfg.CLASSIFICATION_URL)

	repo := repository.NewRepository(db)
	services := services.NewService(cfg, repo, minioClient, imgng)

	engine := gin.Default()

	handlers := handlers.NewHandler(services, cfg, engine)
	handlers.InitRoutes()

	log.Printf("Server starting on port %s", cfg.App.Port)
	if err := engine.Run(":" + cfg.App.Port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
