package services

import (
	"diploma-back/internal/config"
	"diploma-back/internal/models"
	"diploma-back/internal/storage"
	"diploma-back/pkg/imaging"
	"gorm.io/gorm"
)

type AdminService struct {
	config      *config.Config
	db          *gorm.DB
	minIOClient *storage.MinIOClient
	imaging     *imaging.Imaging
}

func (s *AdminService) GetUsersWithJobs() ([]*models.User, error) {
	var users []*models.User
	if err := s.db.Preload("ProcessingJobs").Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}
