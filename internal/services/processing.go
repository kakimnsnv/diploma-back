package services

import (
	"bytes"
	"context"
	"diploma-back/internal/config"
	"diploma-back/internal/models"
	"diploma-back/internal/storage"
	"diploma-back/pkg/imaging"
	"encoding/base64"
	"fmt"
	"mime/multipart"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ProcessingService struct {
	config      *config.Config
	db          *gorm.DB
	minIOClient *storage.MinIOClient
	imaging     *imaging.Imaging
}

func (s *ProcessingService) UploadImage(userID uint, file *multipart.FileHeader, ext string) (*models.ProcessingJob, error) {
	fileStream, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer fileStream.Close()

	uniqueID := uuid.New().String()
	filename := fmt.Sprintf("%s%s", uniqueID, ext)
	objectName := fmt.Sprintf("users/%d/original/%s", userID, filename)
	_, err = s.minIOClient.UploadFile(objectName, fileStream, file.Size, file.Header.Get("Content-Type"))
	if err != nil {
		return nil, err
	}

	// Create a processing job
	job := &models.ProcessingJob{
		UserID:       userID,
		Name:         file.Filename,
		InputNiiPath: objectName,
		Status:       "processing",
	}

	if err := s.db.Create(&job).Error; err != nil {
		return nil, err
	}

	// Process in goroutine
	go s.processImageAsync(userID, uniqueID, job, file)
	return job, nil
}

func (s *ProcessingService) processImageAsync(userID uint, uniqueID string, job *models.ProcessingJob, file *multipart.FileHeader) {
	fileStream, err := file.Open()
	if err != nil {
		job.Status = "failed"
		job.ErrorMessage = fmt.Sprintf("File error: %s", err.Error())
		s.db.Save(job)
		return
	}
	defer fileStream.Close()

	// Call model
	res, err := s.imaging.CallModel(file)
	if err != nil {
		job.Status = "failed"
		job.ErrorMessage = fmt.Sprintf("Model error: %s", err.Error())
		s.db.Save(job)
		return
	}

	decoded, err := base64.StdEncoding.DecodeString(res.ImageBase64)
	if err != nil {
		job.Status = "failed"
		job.ErrorMessage = fmt.Sprintf("Decoding error: %s", err.Error())
		s.db.Save(job)
		return
	}

	objectName := fmt.Sprintf("users/%d/result/%s.png", userID, uniqueID)
	reader := bytes.NewReader(decoded)

	_, err = s.minIOClient.UploadFile(objectName, reader, reader.Size(), "image/png")
	if err != nil {
		job.Status = "failed"
		job.ErrorMessage = fmt.Sprintf("Failed to upload result image: %s", err.Error())
		s.db.Save(job)
		return
	}

	// Update job
	job.OutputImage = objectName
	job.Status = "completed"
	s.db.Save(job)
}

func (s *ProcessingService) GetResult(jobID string, userID uint) (*models.ResultResponse, error) {
	var job models.ProcessingJob
	if err := s.db.Where("id = ? AND user_id = ?", jobID, userID).First(&job).Error; err != nil {
		return nil, err
	}

	response := models.ResultResponse{
		ID:        job.ID,
		Status:    job.Status,
		Name:      job.Name,
		CreatedAt: job.CreatedAt,
		UpdatedAt: job.UpdatedAt,
	}

	if job.Status == "completed" {
		ctx := context.Background()

		if job.OutputImage != "" {
			url, err := s.minIOClient.GetPresignedURL(ctx, job.OutputImage)
			if err != nil {
				fmt.Printf("error: %v", err)
			}
			response.OutputImageURL = url
		}

		if job.InputNiiPath != "" {
			url, err := s.minIOClient.GetPresignedURL(ctx, job.InputNiiPath)
			if err != nil {
				fmt.Printf("error: %v", err)
			}
			response.InputNiiURL = url
		}
	}

	if job.Status == "failed" {
		response.Error = job.ErrorMessage
	}
	return &response, nil
}

func (s *ProcessingService) DeleteResult(jobID string, userID uint) error {
	var job models.ProcessingJob
	if err := s.db.Where("id = ? AND user_id = ?", jobID, userID).First(&job).Error; err != nil {
		return err
	}

	return s.db.Delete(&job).Error
}

func (s *ProcessingService) GetHistory(userID uint) ([]*models.ProcessingJob, error) {
	var jobs []*models.ProcessingJob
	if err := s.db.Where("user_id = ?", userID).Order("created_at DESC").Limit(50).Find(&jobs).Error; err != nil {
		return nil, err
	}

	return jobs, nil
}
