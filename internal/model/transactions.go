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
	CategoryID    *uint          `gorm:"column:category_id" json:"category_id"`
	Amount        float64        `gorm:"type:decimal(12,2);not null" json:"amount"`
	Type          string         `gorm:"type:transaction_type;not null" json:"type"`
	Subtype       *string        `gorm:"type:transaction_subtype" json:"subtype"`
	Description   *string        `gorm:"type:text" json:"description"`
	Date          *time.Time     `gorm:"type:timestamptz" json:"date"`
	Frequency     *string        `gorm:"type:varchar(50)" json:"frequency"`
	StartDate     *time.Time     `gorm:"type:date;column:start_date" json:"start_date"`
	EndDate       *time.Time     `gorm:"type:date;column:end_date" json:"end_date"`
	PrepaidFromID *uint          `gorm:"column:prepaid_from_id" json:"prepaid_from_id"`
	CreatedAt     time.Time      `gorm:"type:timestamptz;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt     time.Time      `gorm:"type:timestamptz;default:CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}

func (Transaction) TableName() string {
	return "transactions"
}
