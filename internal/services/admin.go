package services

import (
	"diploma-back/internal/models"
	"diploma-back/internal/repository"
)

type AdminService struct {
	userRepo *repository.UserRepository
}

func (s *AdminService) GetUsersWithJobs() ([]*models.User, error) {
	return s.userRepo.FindAllWithJobs()
}
