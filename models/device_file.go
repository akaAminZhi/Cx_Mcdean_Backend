// app/models/device_file.go
package models

import (
	"time"

	"gorm.io/gorm"
)

type DeviceFile struct {
	ID       uint   `json:"id" gorm:"primaryKey"`
	DeviceID string `json:"device_id" gorm:"index"` // 对应 Device.ID
	Project  string `json:"project" gorm:"index"`   // 冗余一份方便按项目查
	FileType string `json:"file_type" gorm:"index"` // panel_schedule / test_report / other
	FileName string `json:"file_name"`              // 原始文件名，前端可展示
	FilePath string `json:"file_path"`              // 在服务器上的路径
	FileSize int64  `json:"file_size"`
	MimeType string `json:"mime_type"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}
