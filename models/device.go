package models

import (
	"time"

	"github.com/lib/pq"
	"gorm.io/gorm"
)

type Device struct {
	// 使用你的字符串 id 作为主键
	ID              string        `json:"id" gorm:"primaryKey;size:64"`
	Project         string        `json:"project" gorm:"index"`
	FilePage        int           `json:"file_page" gorm:"index"`
	Subject         string        `json:"subject" gorm:"index"`
	RectPX          pq.Int64Array `json:"rect_px" gorm:"type:integer[]"`
	Text            string        `json:"text" gorm:"index"`
	Comments        string        `json:"comments"`
	Energized       bool          `json:"energized"`
	EnergizedToday  bool          `json:"energized_today"`
	From            string        `json:"from,omitempty"`
	To              string        `json:"to,omitempty"`
	WillEnergizedAt *time.Time    `json:"will_energized_at,omitempty"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}
