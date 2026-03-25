package services

import (
	"errors"

	"diploma-back/internal/auth"
	"diploma-back/internal/config"
	"diploma-back/internal/models"
	"diploma-back/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	config   *config.Config
	userRepo *repository.UserRepository
}

func (s *UserService) CreateUser(req models.RegisterRequest) (*models.AuthResponse, error) {
	if existingUser, _ := s.userRepo.FindByEmail(req.Email); existingUser != nil {
		return nil, errors.New("user already exists")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := models.User{
		Email:    req.Email,
		Password: string(hashedPassword),
		Name:     req.Name,
	}

	if err := s.userRepo.Create(&user); err != nil {
		return nil, err
	}

	token, err := auth.GenerateToken(&s.config.JWT, user.ID, "user")
	if err != nil {
		return nil, err
	}
	resp := models.AuthResponse{Token: token, User: user}
	return &resp, nil
}

func (s *UserService) Login(req models.LoginRequest) (*models.AuthResponse, error) {
	user, err := s.userRepo.FindByEmail(req.Email)
	if err != nil {
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, err
	}

	token, err := auth.GenerateToken(&s.config.JWT, user.ID, user.Role)
	if err != nil {
		return nil, err
	}
	resp := models.AuthResponse{Token: token, User: *user}
	return &resp, nil
}

func (s *UserService) GetProfile(userID uint) (*models.User, error) {
	return s.userRepo.FindByID(userID)
}

func (s *UserService) UpdateProfile(userID uint, req models.UpdateUser) error {
	if _, err := s.userRepo.FindByID(userID); err != nil {
		return err
	}
	return s.userRepo.Update(userID, req)
}
