// internal/models/models.go
package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	Email     string         `gorm:"unique;not null" json:"email"`
	Password  string         `gorm:"not null" json:"-"`
	Name      string         `json:"name"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	Role      string         `gorm:"default:'user'" json:"role"`

	ProcessingJobs []ProcessingJob `gorm:"foreignKey:UserID" json:"processing_jobs,omitempty"`
}

type ProcessingJob struct {
	ID           uint           `gorm:"primarykey" json:"id"`
	UserID       uint           `gorm:"not null" json:"user_id"`
	Name         string         `json:"name"`
	InputNiiPath string         `json:"input_nii_path"`
	OutputImage  string         `json:"output_image"`
	Status       string         `gorm:"default:'pending'" json:"status"` // pending, processing, completed, failed
	ErrorMessage string         `json:"error_message,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`

	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Name     string `json:"name" binding:"required"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type ResultResponse struct {
	ID             uint      `json:"id"`
	Status         string    `json:"status"`
	Name           string    `json:"name"`
	InputNiiURL    string    `json:"input_nii_url"`
	OutputImageURL string    `json:"output_image_url"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`

	Error string `json:"error"`
}
