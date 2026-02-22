package services

import (
	"diploma-back/internal/auth"
	"diploma-back/internal/config"
	"diploma-back/internal/models"
	"diploma-back/internal/storage"
	"diploma-back/pkg/imaging"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserService struct {
	config      *config.Config
	db          *gorm.DB
	minIOClient *storage.MinIOClient
	imaging     *imaging.Imaging
}

func (s *UserService) CreateUser(req models.RegisterRequest) (*models.AuthResponse, error) {
	var existingUser models.User
	if err := s.db.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		return nil, err
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// Create user
	user := models.User{
		Email:    req.Email,
		Password: string(hashedPassword),
		Name:     req.Name,
	}

	if err := s.db.Create(&user).Error; err != nil {
		return nil, err
	}

	// Generate token
	token, err := auth.GenerateToken(&s.config.JWT, user.ID, "user")
	if err != nil {
		return nil, err
	}
	resp := models.AuthResponse{Token: token, User: user}
	return &resp, nil
}

func (s *UserService) Login(req models.LoginRequest) (*models.AuthResponse, error) {
	// Find user
	var user models.User
	if err := s.db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		return nil, err
	}

	// Check password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, err
	}

	// Generate token
	token, err := auth.GenerateToken(&s.config.JWT, user.ID, user.Role)
	if err != nil {
		return nil, err
	}
	resp := models.AuthResponse{Token: token, User: user}
	return &resp, nil
}

func (s *UserService) GetProfile(userID uint) (*models.User, error) {
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return nil, err
	}
	return &user, nil
}
