// internal/handlers/processing.go
package handlers

import (
	"bytes"
	"context"
	"diploma-back/internal/config"
	"diploma-back/internal/middleware"
	"diploma-back/internal/models"
	"diploma-back/internal/storage"
	"diploma-back/pkg/imaging"
	"encoding/base64"
	"fmt"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Handler struct {
	handler     *gin.Engine
	Config      *config.Config
	DB          *gorm.DB
	MinIOClient *storage.MinIOClient
	Imaging     *imaging.Imaging
}

func NewHandler(cfg *config.Config, db *gorm.DB, minioClient *storage.MinIOClient, imgng *imaging.Imaging) *Handler {
	return &Handler{
		Config:      cfg,
		DB:          db,
		MinIOClient: minioClient,
		Imaging:     imgng,
		handler:     gin.Default(),
	}
}

func (h *Handler) InitRoutes() {
	// CORS middleware
	h.handler.Use(middleware.CORSMiddleware())

	// Public routes
	public := h.handler.Group("/api")
	{
		public.POST("/register", h.Register)
		public.POST("/login", h.Login)
		public.POST("/logout", h.Logout)
	}

	// Protected routes
	protected := h.handler.Group("/api")
	protected.Use(middleware.AuthMiddleware(&h.Config.JWT))
	{
		protected.GET("/profile", h.GetProfile)
		protected.POST("/upload", h.UploadImage)
		protected.GET("/results/:id", h.GetResult)
		protected.GET("/history", h.GetHistory)
	}
}

func (h *Handler) Start() error {
	return h.handler.Run(":" + h.Config.App.Port)
}

func (h *Handler) UploadImage(c *gin.Context) {
	userID := c.GetUint("userID")

	// Get file
	file, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No image file provided"})
		return
	}

	// Validate file type
	ext := filepath.Ext(file.Filename)
	if ext != ".nii" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only NII files are allowed"})
		return
	}

	fileStream, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unexpected error"})
		return
	}
	defer fileStream.Close()

	// Upload to MinIO
	uniqueID := uuid.New().String()
	filename := fmt.Sprintf("%s%s", uniqueID, ext)
	objectName := fmt.Sprintf("users/%d/original/%s", userID, filename)
	_, err = h.MinIOClient.UploadFile(objectName, fileStream, file.Size, file.Header.Get("Content-Type"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload to minio"})
		return
	}

	// Create processing job
	job := &models.ProcessingJob{
		UserID:       userID,
		InputNiiPath: objectName,
		Status:       "processing",
	}

	if err := h.DB.Create(&job).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create processing job"})
		return
	}

	// Process in goroutine
	go h.processImageAsync(userID, uniqueID, job, file)

	c.JSON(http.StatusOK, gin.H{
		"message": "Processing started",
		"job_id":  job.ID,
		"status":  "processing",
	})
}

func (h *Handler) processImageAsync(userID uint, uniqueID string, job *models.ProcessingJob, file *multipart.FileHeader) {
	fileStream, err := file.Open()
	if err != nil {
		job.Status = "failed"
		job.ErrorMessage = fmt.Sprintf("File error: %s", err.Error())
		h.DB.Save(job)
		return
	}
	defer fileStream.Close()

	// Call model
	res, err := h.Imaging.CallModel(file)
	if err != nil {
		job.Status = "failed"
		job.ErrorMessage = fmt.Sprintf("Model error: %s", err.Error())
		h.DB.Save(job)
		return
	}

	decoded, err := base64.StdEncoding.DecodeString(res.ImageBase64)
	if err != nil {
		job.Status = "failed"
		job.ErrorMessage = fmt.Sprintf("Decoding error: %s", err.Error())
		h.DB.Save(job)
		return
	}

	objectName := fmt.Sprintf("users/%d/result/%s.png", userID, uniqueID)
	reader := bytes.NewReader(decoded)

	_, err = h.MinIOClient.UploadFile(objectName, reader, reader.Size(), "image/png")
	if err != nil {
		job.Status = "failed"
		job.ErrorMessage = fmt.Sprintf("Failed to upload result image: %s", err.Error())
		h.DB.Save(job)
		return
	}

	// Update job
	job.OutputImage = objectName
	job.Status = "completed"
	h.DB.Save(job)
}

type ResultResponse struct {
	ID             uint      `json:"id"`
	Status         string    `json:"status"`
	InputNiiURL    string    `json:"input_nii_url"`
	OutputImageURL string    `json:"output_image_url"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`

	Error string `json:"error"`
}

func (h *Handler) GetResult(c *gin.Context) {
	jobID := c.Param("id")
	userID := c.GetUint("userID")

	var job models.ProcessingJob
	if err := h.DB.Where("id = ? AND user_id = ?", jobID, userID).First(&job).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}

	response := ResultResponse{
		ID:        job.ID,
		Status:    job.Status,
		CreatedAt: job.CreatedAt,
		UpdatedAt: job.UpdatedAt,
	}

	if job.Status == "completed" {
		ctx := context.Background()

		if job.OutputImage != "" {
			url, err := h.MinIOClient.GetPresignedURL(ctx, job.OutputImage)
			if err != nil {
				fmt.Printf("error: %v", err)
			}
			response.OutputImageURL = url
		}

		if job.InputNiiPath != "" {
			url, err := h.MinIOClient.GetPresignedURL(ctx, job.InputNiiPath)
			if err != nil {
				fmt.Printf("error: %v", err)
			}
			response.InputNiiURL = url
		}
	}

	if job.Status == "failed" {
		response.Error = job.ErrorMessage
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) GetHistory(c *gin.Context) {
	userID := c.GetUint("userID")

	var jobs []*models.ProcessingJob
	if err := h.DB.Where("user_id = ?", userID).Order("created_at DESC").Limit(50).Find(&jobs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch history"})
		return
	}

	c.JSON(http.StatusOK, jobs)
}
