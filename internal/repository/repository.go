package repository

import "gorm.io/gorm"

type Repository struct {
	User           *UserRepository
	Job            *JobRepository
	Classification *ClassificationRepository
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{
		User:           NewUserRepository(db),
		Job:            NewJobRepository(db),
		Classification: NewClassificationRepository(db),
	}
}
