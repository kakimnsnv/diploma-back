package services

import (
	"diploma-back/internal/config"
	"diploma-back/internal/storage"
	"diploma-back/pkg/imaging"
	"gorm.io/gorm"
)

type Service struct {
	UserService       *UserService
	ProcessingService *ProcessingService
	AdminService      *AdminService
}

func NewService(config *config.Config, db *gorm.DB, minIOClient *storage.MinIOClient, imaging *imaging.Imaging) *Service {
	userService := &UserService{
		db:          db,
		config:      config,
		minIOClient: minIOClient,
		imaging:     imaging,
	}

	processingService := &ProcessingService{
		db:          db,
		config:      config,
		minIOClient: minIOClient,
		imaging:     imaging,
	}

	adminService := &AdminService{
		db:          db,
		config:      config,
		minIOClient: minIOClient,
		imaging:     imaging,
	}

	return &Service{
		UserService:       userService,
		ProcessingService: processingService,
		AdminService:      adminService,
	}
}
