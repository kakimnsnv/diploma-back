// cmd/server/main.go
package main

import (
	"diploma-back/internal/database"
	"diploma-back/internal/handlers"
	"diploma-back/internal/middleware"
	"diploma-back/internal/storage"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Initialize database
	db, err := database.InitDB()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Auto-migrate models
	if err := database.MigrateDB(db); err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	minioClient, err := storage.NewMinIOClient()
	if err != nil {
		log.Fatal("Failed to initialize MinIO client:", err)
	}

	// Initialize Gin router
	r := gin.Default()

	// CORS middleware
	r.Use(middleware.CORSMiddleware())

	// Public routes
	public := r.Group("/api")
	{
		public.POST("/register", handlers.Register(db))
		public.POST("/login", handlers.Login(db))
		public.POST("/logout", handlers.Logout)
	}

	// Protected routes
	protected := r.Group("/api")
	protected.Use(middleware.AuthMiddleware())
	{
		protected.GET("/profile", handlers.GetProfile(db))
		protected.POST("/upload", handlers.UploadImage(db, minioClient))
		// protected.POST("/process", handlers.ProcessImage(db))
		protected.GET("/results/:id", handlers.GetResult(db, minioClient))
		protected.GET("/results/:id/download", handlers.DownloadResult(db, minioClient))
		protected.GET("/history", handlers.GetHistory(db, minioClient))
	}

	// Get port from env or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
