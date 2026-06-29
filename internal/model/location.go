package model

import (
	"time"

	"gorm.io/gorm"
)

type Location struct {
	ID             uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	CreatedAt      time.Time      `gorm:"column:created_at" json:"created_at"`
	UpdatedAt      time.Time      `gorm:"column:updated_at" json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"deleted_at"`
	Name           string         `gorm:"column:name;not null;size:255" json:"name"`
	NormalizedName string         `gorm:"column:normalized_name;not null;size:255" json:"-"`
}

func (Location) TableName() string {
	return "locations"
}
