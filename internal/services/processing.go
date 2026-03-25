package services

import (
	"bytes"
	"context"
	"diploma-back/internal/config"
	"diploma-back/internal/models"
	"diploma-back/internal/repository"
	"diploma-back/internal/storage"
	"diploma-back/pkg/imaging"
	"encoding/base64"
	"fmt"
	"mime/multipart"

	"github.com/google/uuid"
)

type ProcessingService struct {
	config      *config.Config
	jobRepo     *repository.JobRepository
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

	job := &models.ProcessingJob{
		UserID:       userID,
		Name:         file.Filename,
		InputNiiPath: objectName,
		Status:       "processing",
	}

	if err := s.jobRepo.Create(job); err != nil {
		return nil, err
	}

	go s.processImageAsync(userID, uniqueID, job, file)
	return job, nil
}

func (s *ProcessingService) processImageAsync(userID uint, uniqueID string, job *models.ProcessingJob, file *multipart.FileHeader) {
	fileStream, err := file.Open()
	if err != nil {
		job.Status = "failed"
		job.ErrorMessage = fmt.Sprintf("File error: %s", err.Error())
		s.jobRepo.Save(job)
		return
	}
	defer fileStream.Close()

	res, err := s.imaging.CallModel(file)
	if err != nil {
		job.Status = "failed"
		job.ErrorMessage = fmt.Sprintf("Model error: %s", err.Error())
		s.jobRepo.Save(job)
		return
	}

	decoded, err := base64.StdEncoding.DecodeString(res.ImageBase64)
	if err != nil {
		job.Status = "failed"
		job.ErrorMessage = fmt.Sprintf("Decoding error: %s", err.Error())
		s.jobRepo.Save(job)
		return
	}

	objectName := fmt.Sprintf("users/%d/result/%s.png", userID, uniqueID)
	reader := bytes.NewReader(decoded)

	_, err = s.minIOClient.UploadFile(objectName, reader, reader.Size(), "image/png")
	if err != nil {
		job.Status = "failed"
		job.ErrorMessage = fmt.Sprintf("Failed to upload result image: %s", err.Error())
		s.jobRepo.Save(job)
		return
	}

	job.OutputImage = objectName
	job.Status = "completed"
	s.jobRepo.Save(job)
}

func (s *ProcessingService) GetResult(jobID string, userID uint) (*models.ResultResponse, error) {
	job, err := s.jobRepo.FindByIDAndUserID(jobID, userID)
	if err != nil {
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
	job, err := s.jobRepo.FindByIDAndUserID(jobID, userID)
	if err != nil {
		return err
	}
	return s.jobRepo.Delete(job)
}

func (s *ProcessingService) GetHistory(userID uint) ([]*models.ProcessingJob, error) {
	return s.jobRepo.FindByUserID(userID, 50)
}
