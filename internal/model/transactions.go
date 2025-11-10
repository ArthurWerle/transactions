package model

import (
	"time"
)

type Transactions struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	MigratedID  *uint      `gorm:"column:migrated_id" json:"migrated_id,omitempty"`
	IsRecurring bool       `gorm:"not null;default:false" json:"is_recurring"`
	CategoryID  *uint      `gorm:"column:category_id" json:"category_id,omitempty"`
	Amount      float64    `gorm:"type:decimal(12,2);not null" json:"amount"` // Consider using decimal.Decimal for precision
	Type        string     `gorm:"type:transaction_type;not null" json:"type"`
	Subtype     *string    `gorm:"type:transaction_subtype" json:"subtype,omitempty"`
	Description *string    `gorm:"type:text" json:"description,omitempty"`
	Date        *time.Time `gorm:"type:timestamptz" json:"date,omitempty"`
	Frequency   *string    `gorm:"type:varchar(50)" json:"frequency,omitempty"`
	StartDate   *time.Time `gorm:"type:date;column:start_date" json:"start_date,omitempty"`
	EndDate     *time.Time `gorm:"type:date;column:end_date" json:"end_date,omitempty"`
	CreatedAt   time.Time  `gorm:"type:timestamptz;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt   time.Time  `gorm:"type:timestamptz;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

// TableName specifies the table name for the Transactions model
func (Transactions) TableName() string {
	return "transactions_v2"
}
