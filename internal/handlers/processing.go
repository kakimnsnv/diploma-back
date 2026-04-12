package handlers

import (
	"diploma-back/internal/middleware"
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

func (h *Handler) InitProcessingRoutes() {
	// CORS middleware
	h.handler.Use(middleware.CORSMiddleware())

	// Protected routes
	protected := h.handler.Group("/api")
	protected.Use(middleware.AuthMiddleware(&h.Config.JWT))
	{
		protected.GET("/profile", h.GetProfile)
		protected.POST("/upload", h.UploadImage)
		protected.GET("/results/:id", h.GetResult)
		protected.POST("/results/:id/classify", h.ClassifyImage)
		protected.POST("/results/:id/segment", h.SegmentJob)
		protected.DELETE("/results/:id", h.DeleteResult)
		protected.POST("/classify", h.ClassifyUpload)
		protected.POST("/slices", h.GetSlices)
		protected.GET("/history", h.GetHistory)
	}
}

func (h *Handler) UploadImage(c *gin.Context) {
	userID := c.GetUint("userID")

	// Get a file
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

	job, err := h.Services.ProcessingService.UploadImage(userID, file, ext)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload image"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Processing started",
		"job_id":  job.ID,
		"status":  "processing",
	})
}

func (h *Handler) GetResult(c *gin.Context) {
	jobID := c.Param("id")
	userID := c.GetUint("userID")

	response, err := h.Services.ProcessingService.GetResult(jobID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch result"})
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) ClassifyImage(c *gin.Context) {
	jobID := c.Param("id")
	userID := c.GetUint("userID")

	result, err := h.Services.ProcessingService.ClassifyResult(jobID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) SegmentJob(c *gin.Context) {
	jobID := c.Param("id")
	userID := c.GetUint("userID")

	file, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file provided"})
		return
	}

	ext := filepath.Ext(file.Filename)
	if ext != ".nii" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only NII files are allowed"})
		return
	}

	job, err := h.Services.ProcessingService.SegmentJob(jobID, userID, file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Segmentation started",
		"job_id":  job.ID,
		"status":  "processing",
	})
}

func (h *Handler) ClassifyUpload(c *gin.Context) {
	userID := c.GetUint("userID")

	file, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No image file provided"})
		return
	}

	ext := filepath.Ext(file.Filename)
	if ext != ".png" && ext != ".jpg" && ext != ".jpeg" && ext != ".tiff" && ext != ".nii" && ext != ".nii.gz" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only PNG, JPG, TIFF, and NIfTI files are allowed"})
		return
	}

	result, err := h.Services.ProcessingService.ClassifyUpload(userID, file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) DeleteResult(c *gin.Context) {
	jobID := c.Param("id")
	userID := c.GetUint("userID")

	err := h.Services.ProcessingService.DeleteResult(jobID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete result"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Result deleted successfully"})
}

func (h *Handler) GetSlices(c *gin.Context) {
	userID := c.GetUint("userID")

	file, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file provided"})
		return
	}

	ext := filepath.Ext(file.Filename)
	if ext != ".nii" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only NII files are allowed"})
		return
	}

	result, err := h.Services.ProcessingService.GetSlices(userID, file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) GetHistory(c *gin.Context) {
	userID := c.GetUint("userID")

	jobs, err := h.Services.ProcessingService.GetHistory(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch history"})
		return
	}

	c.JSON(http.StatusOK, jobs)
}
