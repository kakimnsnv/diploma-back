package services

import (
	"diploma-back/internal/config"
	"diploma-back/internal/repository"
	"diploma-back/internal/storage"
	"diploma-back/pkg/imaging"
)

type Service struct {
	UserService       *UserService
	ProcessingService *ProcessingService
	AdminService      *AdminService
}

func NewService(config *config.Config, repo *repository.Repository, minIOClient *storage.MinIOClient, imaging *imaging.Imaging) *Service {
	userService := &UserService{
		config:   config,
		userRepo: repo.User,
	}

	processingService := &ProcessingService{
		config:             config,
		jobRepo:            repo.Job,
		classificationRepo: repo.Classification,
		minIOClient:        minIOClient,
		imaging:            imaging,
	}

	adminService := &AdminService{
		userRepo: repo.User,
	}

	return &Service{
		UserService:       userService,
		ProcessingService: processingService,
		AdminService:      adminService,
	}
}