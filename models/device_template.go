package models

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type DeviceTemplate struct {
	ID      uint           `json:"id" gorm:"primaryKey"`
	Name    string         `json:"name" gorm:"index"`
	Subject string         `json:"subject" gorm:"index"`
	Steps   datatypes.JSON `json:"steps" gorm:"type:jsonb"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}
