package repository

import (
	"diploma-back/internal/models"

	"gorm.io/gorm"
)

type ClassificationRepository struct {
	db *gorm.DB
}

func NewClassificationRepository(db *gorm.DB) *ClassificationRepository {
	return &ClassificationRepository{db: db}
}

func (r *ClassificationRepository) Create(result *models.ClassificationResult) error {
	return r.db.Create(result).Error
}

func (r *ClassificationRepository) FindByJobID(jobID uint) (*models.ClassificationResult, error) {
	var result models.ClassificationResult
	if err := r.db.Where("processing_job_id = ?", jobID).First(&result).Error; err != nil {
		return nil, err
	}
	return &result, nil
}
