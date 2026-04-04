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
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"

	"github.com/google/uuid"
)

type ProcessingService struct {
	config             *config.Config
	jobRepo            *repository.JobRepository
	classificationRepo *repository.ClassificationRepository
	minIOClient        *storage.MinIOClient
	imaging            *imaging.Imaging
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

	// Listing 2.X — Non-blocking inference dispatch
	job := &models.ProcessingJob{
		UserID:       userID,
		InputNiiPath: objectName,
		Status:       "processing",
	}
	s.jobRepo.Create(job) // persist before the goroutine starts

	go s.processImageAsync(userID, uniqueID, job, file)
	// HTTP handler returns job.ID immediately
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

func (s *ProcessingService) SegmentJob(jobID string, userID uint, file *multipart.FileHeader) (*models.ProcessingJob, error) {
	job, err := s.jobRepo.FindByIDAndUserID(jobID, userID)
	if err != nil {
		return nil, fmt.Errorf("job not found: %w", err)
	}

	if job.InputNiiPath != "" {
		return nil, fmt.Errorf("job already has segmentation input")
	}

	fileStream, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer fileStream.Close()

	uniqueID := uuid.New().String()
	ext := filepath.Ext(file.Filename)
	filename := fmt.Sprintf("%s%s", uniqueID, ext)
	objectName := fmt.Sprintf("users/%d/original/%s", userID, filename)
	_, err = s.minIOClient.UploadFile(objectName, fileStream, file.Size, file.Header.Get("Content-Type"))
	if err != nil {
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	job.InputNiiPath = objectName
	job.Status = "processing"
	s.jobRepo.Save(job)

	go s.processImageAsync(userID, uniqueID, job, file)

	return job, nil
}

func (s *ProcessingService) ClassifyResult(jobID string, userID uint) (*models.ClassificationResult, error) {
	job, err := s.jobRepo.FindByIDAndUserID(jobID, userID)
	if err != nil {
		return nil, fmt.Errorf("job not found: %w", err)
	}

	if job.Status != "completed" {
		return nil, fmt.Errorf("job is not completed, current status: %s", job.Status)
	}

	if job.OutputImage == "" {
		return nil, fmt.Errorf("job has no output image")
	}

	imageBytes, err := s.minIOClient.DownloadFile(job.OutputImage)
	if err != nil {
		return nil, fmt.Errorf("failed to download output image: %w", err)
	}

	filename := filepath.Base(job.OutputImage)
	classRes, err := s.imaging.CallClassifier(imageBytes, filename)
	if err != nil {
		return nil, fmt.Errorf("classification failed: %w", err)
	}

	probsJSON, _ := json.Marshal(classRes.ClassProbabilities)
	namesJSON, _ := json.Marshal(classRes.ClassNames)

	result := &models.ClassificationResult{
		ProcessingJobID:    job.ID,
		PredictedClass:     classRes.PredictedClass,
		PredictedClassName: classRes.PredictedClassName,
		Confidence:         classRes.Confidence,
		ClassProbabilities: string(probsJSON),
		ClassNames:         string(namesJSON),
	}

	if err := s.classificationRepo.Create(result); err != nil {
		return nil, fmt.Errorf("failed to save classification result: %w", err)
	}

	job.ClassificationResultID = &result.ID
	if err := s.jobRepo.Save(job); err != nil {
		return nil, fmt.Errorf("failed to update job: %w", err)
	}

	return result, nil
}

func (s *ProcessingService) ClassifyUpload(userID uint, file *multipart.FileHeader) (*models.ProcessingJob, error) {
	fileStream, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer fileStream.Close()

	fileBytes, err := io.ReadAll(fileStream)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Upload image to MinIO
	uniqueID := uuid.New().String()
	ext := filepath.Ext(file.Filename)
	objectName := fmt.Sprintf("users/%d/classification/%s%s", userID, uniqueID, ext)
	reader := bytes.NewReader(fileBytes)
	_, err = s.minIOClient.UploadFile(objectName, reader, int64(len(fileBytes)), file.Header.Get("Content-Type"))
	if err != nil {
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	// Create processing job
	job := &models.ProcessingJob{
		UserID:      userID,
		Name:        file.Filename,
		OutputImage: objectName,
		Status:      "completed",
	}
	if err := s.jobRepo.Create(job); err != nil {
		return nil, fmt.Errorf("failed to create job: %w", err)
	}

	// Call classifier
	classRes, err := s.imaging.CallClassifier(fileBytes, file.Filename)
	if err != nil {
		job.Status = "failed"
		job.ErrorMessage = fmt.Sprintf("Classification failed: %s", err.Error())
		s.jobRepo.Save(job)
		return nil, fmt.Errorf("classification failed: %w", err)
	}

	probsJSON, _ := json.Marshal(classRes.ClassProbabilities)
	namesJSON, _ := json.Marshal(classRes.ClassNames)

	result := &models.ClassificationResult{
		ProcessingJobID:    job.ID,
		PredictedClass:     classRes.PredictedClass,
		PredictedClassName: classRes.PredictedClassName,
		Confidence:         classRes.Confidence,
		ClassProbabilities: string(probsJSON),
		ClassNames:         string(namesJSON),
	}

	if err := s.classificationRepo.Create(result); err != nil {
		return nil, fmt.Errorf("failed to save classification result: %w", err)
	}

	job.ClassificationResultID = &result.ID
	s.jobRepo.Save(job)

	return job, nil
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

	if job.ClassificationResultID != nil {
		classResult, err := s.classificationRepo.FindByJobID(job.ID)
		if err == nil {
			response.ClassificationResult = classResult
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
