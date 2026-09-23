package model

import (
	"time"

	"gorm.io/gorm"
)

type Category struct {
	ID          uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	CreatedAt   time.Time      `gorm:"column:created_at" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"column:updated_at" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"deleted_at"`
	Name        string         `gorm:"column:name;not null;size:255" json:"name"`
	Description string         `gorm:"column:description;type:text" json:"description"`
	Color       string         `gorm:"column:color;size:50" json:"color"`
	// ExcludeFromCalculations keeps the category's transactions out of every
	// aggregate (averages, reports, percentages) while still listing them.
	ExcludeFromCalculations bool `gorm:"column:exclude_from_calculations;not null;default:false" json:"exclude_from_calculations"`
}

func (Category) TableName() string {
	return "categories"
}
