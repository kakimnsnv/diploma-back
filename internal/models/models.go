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

	ProcessingJobs []ProcessingJob `gorm:"foreignKey:UserID" json:"processing_jobs,omitempty"`
}

type ProcessingJob struct {
	ID               uint           `gorm:"primarykey" json:"id"`
	UserID           uint           `gorm:"not null" json:"user_id"`
	InputNiiPath     string         `json:"input_nii_path"`
	OutputNiiPath    string         `json:"output_nii_path"`
	OriginalImageURL string         `json:"original_image_url" gorm:"original_image_url"`
	ResultImageURL   string         `json:"result_image_url" gorm:"result_image_url"`
	Status           string         `gorm:"default:'pending'" json:"status"` // pending, processing, completed, failed
	ErrorMessage     string         `json:"error_message,omitempty"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`

	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}
