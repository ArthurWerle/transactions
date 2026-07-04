package model

import (
	"time"

	"gorm.io/gorm"
)

type Transaction struct {
	ID            uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	MigratedID    *uint          `gorm:"column:migrated_id" json:"migrated_id"`
	CreatedById   uint           `gorm:"column:created_by_id" json:"created_by_id"`
	IsRecurring   bool           `gorm:"not null;default:false" json:"is_recurring"`
	CategoryID    uint           `gorm:"column:category_id;not null" json:"category_id"`
	SubcategoryID *uint          `gorm:"column:subcategory_id" json:"subcategory_id"`
	Subcategory   *Subcategory   `gorm:"foreignKey:SubcategoryID" json:"subcategory,omitempty"`
	LocationID    *uint          `gorm:"column:location_id" json:"location_id"`
	Location      *Location      `gorm:"foreignKey:LocationID" json:"location,omitempty"`
	Amount        float64        `gorm:"type:decimal(12,2);not null" json:"amount"`
	Type          string         `gorm:"type:transaction_type;not null" json:"type"`
	// Deprecated: Subtype will be removed. Do not use.
	Subtype       *string        `gorm:"type:transaction_subtype" json:"subtype,omitempty"`
	Origin        string         `gorm:"type:transaction_origin;not null;default:web" json:"origin"`
	Description   *string        `gorm:"type:text" json:"description"`
	Date          *time.Time     `gorm:"type:timestamptz" json:"date"`
	Frequency     *string        `gorm:"type:transaction_frequency" json:"frequency"`
	StartDate     *time.Time     `gorm:"type:date;column:start_date" json:"start_date"`
	EndDate       *time.Time     `gorm:"type:date;column:end_date" json:"end_date"`
	PrepaidFromID *uint          `gorm:"column:prepaid_from_id" json:"prepaid_from_id"`
	IsPrepaid     bool           `gorm:"->" json:"is_prepaid"`
	CreatedAt     time.Time      `gorm:"type:timestamptz;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt     time.Time      `gorm:"type:timestamptz;default:CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}

func (Transaction) TableName() string {
	return "transactions"
}
