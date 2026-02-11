// internal/handlers/processing.go
package handlers

import (
	"context"
	"diploma-back/internal/models"
	"diploma-back/internal/storage"
	"diploma-back/pkg/imaging"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func UploadImage(db *gorm.DB, minioClient *storage.MinIOClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetUint("userID")

		file, err := c.FormFile("image")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No image file provided"})
			return
		}

		// Validate file type
		ext := filepath.Ext(file.Filename)
		if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Only JPEG and PNG files are allowed"})
			return
		}

		// Generate unique filename
		uniqueID := uuid.New().String()
		filename := fmt.Sprintf("%s%s", uniqueID, ext)
		tempPath := filepath.Join("/tmp", filename)

		// Save file temporarily
		if err := c.SaveUploadedFile(file, tempPath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
			return
		}
		defer os.Remove(tempPath) // Clean up temp file

		// Validate image
		if err := imaging.ValidateImageFile(tempPath); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid image: %s", err.Error())})
			return
		}

		// Upload to MinIO
		ctx := context.Background()
		objectName := fmt.Sprintf("users/%d/original/%s", userID, filename)

		contentType := "image/jpeg"
		if ext == ".png" {
			contentType = "image/png"
		}

		_, err = minioClient.UploadFile(ctx, objectName, tempPath, contentType)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload to storage"})
			return
		}

		// Create processing job
		job := &models.ProcessingJob{
			UserID:           userID,
			OriginalImageURL: objectName,
			Status:           "processing",
		}

		if err := db.Create(&job).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create processing job"})
			return
		}

		// Process in goroutine
		go processImageAsync(db, job, minioClient)

		c.JSON(http.StatusOK, gin.H{
			"message": "Processing started",
			"job_id":  job.ID,
			"status":  "processing",
		})
	}
}

// func ProcessImage(db *gorm.DB) gin.HandlerFunc {
// 	return func(c *gin.Context) {
// 		var req struct {
// 			JobID uint `json:"job_id" binding:"required"`
// 		}

// 		if err := c.ShouldBindJSON(&req); err != nil {
// 			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 			return
// 		}

// 		userID := c.GetUint("userID")

// 		// Get job
// 		var job models.ProcessingJob
// 		if err := db.Where("id = ? AND user_id = ?", req.JobID, userID).First(&job).Error; err != nil {
// 			c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
// 			return
// 		}

// 		if job.Status != "uploaded" && job.Status != "failed" {
// 			c.JSON(http.StatusBadRequest, gin.H{"error": "Job already processing or completed"})
// 			return
// 		}

// 		// Update status
// 		job.Status = "processing"
// 		db.Save(&job)

// 		// Process in goroutine
// 		go processImageAsync(db, &job)

// 		c.JSON(http.StatusOK, gin.H{
// 			"message": "Processing started",
// 			"job_id":  job.ID,
// 			"status":  "processing",
// 		})
// 	}
// }

func processImageAsync(db *gorm.DB, job *models.ProcessingJob, minioClient *storage.MinIOClient) {
	ctx := context.Background()

	// Download original image from MinIO
	tempImagePath := filepath.Join("/tmp", fmt.Sprintf("img_%d_%s", job.ID, uuid.New().String()))
	err := minioClient.DownloadFile(ctx, job.OriginalImageURL, tempImagePath)
	if err != nil {
		job.Status = "failed"
		job.ErrorMessage = fmt.Sprintf("Failed to download image: %s", err.Error())
		db.Save(job)
		return
	}
	defer os.Remove(tempImagePath)

	// Convert image to NII
	inputNiiPath, err := imaging.ConvertToNii(tempImagePath)
	if err != nil {
		job.Status = "failed"
		job.ErrorMessage = fmt.Sprintf("Conversion error: %s", err.Error())
		db.Save(job)
		return
	}
	defer os.Remove(inputNiiPath)

	// Upload input NII to MinIO
	inputNiiObjectName := fmt.Sprintf("users/%d/input/%s.nii", job.UserID, uuid.New().String())
	_, err = minioClient.UploadFile(ctx, inputNiiObjectName, inputNiiPath, "application/octet-stream")
	if err != nil {
		job.Status = "failed"
		job.ErrorMessage = fmt.Sprintf("Failed to upload input NII: %s", err.Error())
		db.Save(job)
		return
	}

	job.InputNiiPath = inputNiiObjectName
	db.Save(job)

	// Call model
	outputNiiPath, err := imaging.CallModel(inputNiiPath)
	if err != nil {
		job.Status = "failed"
		job.ErrorMessage = fmt.Sprintf("Model error: %s", err.Error())
		db.Save(job)
		return
	}
	defer os.Remove(outputNiiPath)

	// Upload output NII to MinIO
	outputNiiObjectName := fmt.Sprintf("users/%d/output/%s.nii", job.UserID, uuid.New().String())
	_, err = minioClient.UploadFile(ctx, outputNiiObjectName, outputNiiPath, "application/octet-stream")
	if err != nil {
		job.Status = "failed"
		job.ErrorMessage = fmt.Sprintf("Failed to upload output NII: %s", err.Error())
		db.Save(job)
		return
	}

	pngPath, err := imaging.ConvertNiiToImage(outputNiiPath, "png")
	if err != nil {
		job.Status = "failed"
		job.ErrorMessage = fmt.Sprintf("Failed to convert NII to PNG: %s", err.Error())
		db.Save(job)
		return
	}
	defer os.Remove(pngPath)

	outputPNGObjectName := fmt.Sprintf("users/%d/outputPNG/%s.png", job.UserID, uuid.New().String())
	_, err = minioClient.UploadFile(ctx, outputPNGObjectName, pngPath, "application/octet-stream")
	if err != nil {
		job.Status = "failed"
		job.ErrorMessage = fmt.Sprintf("Failed to upload output PNG: %s", err.Error())
		db.Save(job)
		return
	}

	// Update job
	job.OutputNiiPath = outputNiiObjectName
	job.ResultImageURL = outputPNGObjectName
	job.Status = "completed"
	db.Save(job)
}

func GetResult(db *gorm.DB, minioClient *storage.MinIOClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		jobID := c.Param("id")
		userID := c.GetUint("userID")

		var job models.ProcessingJob
		if err := db.Where("id = ? AND user_id = ?", jobID, userID).First(&job).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
			return
		}

		response := gin.H{
			"id":         job.ID,
			"status":     job.Status,
			"created_at": job.CreatedAt,
			"updated_at": job.UpdatedAt,
		}

		if job.Status == "completed" {
			ctx := context.Background()

			if job.ResultImageURL != "" {
				url, err := minioClient.GetPresignedURL(ctx, job.ResultImageURL)
				if err != nil {
					fmt.Printf("error: %v", err)
				}
				response["result_image_url"] = url
			}

			if job.OriginalImageURL != "" {
				url, err := minioClient.GetPresignedURL(ctx, job.OriginalImageURL)
				if err != nil {
					fmt.Printf("error: %v", err)
				}
				response["original_image_url"] = url
			}
		}

		if job.Status == "failed" {
			response["error"] = job.ErrorMessage
		}

		c.JSON(http.StatusOK, response)
	}
}

func GetHistory(db *gorm.DB, minioClient *storage.MinIOClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetUint("userID")
		ctx := context.Background()

		var jobs []*models.ProcessingJob
		if err := db.Where("user_id = ?", userID).Order("created_at DESC").Limit(50).Find(&jobs).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch history"})
			return
		}

		for _, job := range jobs {
			if job.ResultImageURL != "" {
				url, err := minioClient.GetPresignedURL(ctx, job.ResultImageURL)
				if err != nil {
					fmt.Printf("error: %v", err)
				}
				job.ResultImageURL = url
			}

			if job.OriginalImageURL != "" {
				url, err := minioClient.GetPresignedURL(ctx, job.OriginalImageURL)
				if err != nil {
					fmt.Printf("error: %v", err)
				}
				job.OriginalImageURL = url
			}
		}

		c.JSON(http.StatusOK, jobs)
	}
}

func DownloadResult(db *gorm.DB, minioClient *storage.MinIOClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		jobID := c.Param("id")
		userID := c.GetUint("userID")
		format := c.DefaultQuery("format", "nii") // nii or png

		var job models.ProcessingJob
		if err := db.Where("id = ? AND user_id = ?", jobID, userID).First(&job).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
			return
		}

		if job.Status != "completed" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Job not completed"})
			return
		}

		ctx := context.Background()

		if format == "png" {
			// Download NII, convert to PNG, serve
			tempNiiPath := filepath.Join("/tmp", fmt.Sprintf("nii_%s.nii", uuid.New().String()))
			err := minioClient.DownloadFile(ctx, job.OutputNiiPath, tempNiiPath)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to download result"})
				return
			}
			defer os.Remove(tempNiiPath)

			pngPath, err := imaging.ConvertNiiToImage(tempNiiPath, "png")
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert to PNG"})
				return
			}
			defer os.Remove(pngPath)

			c.File(pngPath)
		} else {
			// Serve NII directly
			obj, err := minioClient.GetObject(ctx, job.OutputNiiPath)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get result"})
				return
			}
			defer obj.Close()

			c.Header("Content-Type", "application/octet-stream")
			c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=result_%d.nii", job.ID))
			c.DataFromReader(http.StatusOK, -1, "application/octet-stream", obj, nil)
		}
	}
}
