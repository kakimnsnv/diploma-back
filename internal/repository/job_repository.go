package repository

import (
	"diploma-back/internal/models"

	"gorm.io/gorm"
)

type JobRepository struct {
	db *gorm.DB
}

func NewJobRepository(db *gorm.DB) *JobRepository {
	return &JobRepository{db: db}
}

func (r *JobRepository) Create(job *models.ProcessingJob) error {
	return r.db.Create(job).Error
}

func (r *JobRepository) Save(job *models.ProcessingJob) error {
	return r.db.Save(job).Error
}

func (r *JobRepository) FindByIDAndUserID(jobID string, userID uint) (*models.ProcessingJob, error) {
	var job models.ProcessingJob
	if err := r.db.Where("id = ? AND user_id = ?", jobID, userID).First(&job).Error; err != nil {
		return nil, err
	}
	return &job, nil
}

func (r *JobRepository) Delete(job *models.ProcessingJob) error {
	return r.db.Delete(job).Error
}

func (r *JobRepository) FindByUserID(userID uint, limit int) ([]*models.ProcessingJob, error) {
	var jobs []*models.ProcessingJob
	if err := r.db.Where("user_id = ?", userID).Order("created_at DESC").Limit(limit).Find(&jobs).Error; err != nil {
		return nil, err
	}
	return jobs, nil
}
