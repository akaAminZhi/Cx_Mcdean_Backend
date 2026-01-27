package models

import (
	"time"

	"gorm.io/datatypes"
)

type DeviceSubjectStep struct {
	ID           uint           `json:"id" gorm:"primaryKey"`
	Subject      string         `json:"subject" gorm:"size:64;index;uniqueIndex:idx_subject_key"`
	Key          string         `json:"key" gorm:"size:64;index;uniqueIndex:idx_subject_key"`
	Label        string         `json:"label" gorm:"size:128"`
	StepOrder    int            `json:"step_order" gorm:"index"`
	IsActive     bool           `json:"is_active" gorm:"default:true"`
	Requirements datatypes.JSON `json:"requirements" gorm:"type:jsonb"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
