package repository

import (
	"github.com/ArthurWerle/transactions/internal/model"
	"gorm.io/gorm"
)

type LocationRepository interface {
	Create(location *model.Location) error
	FindByID(id uint) (*model.Location, error)
	FindByNormalizedName(normalizedName string) (*model.Location, error)
	FindAll() ([]model.Location, error)
	Update(location *model.Location) error
	Delete(id uint) error
	Merge(sourceID, targetID uint) error
}

type locationRepository struct {
	db *gorm.DB
}

func NewLocationRepository(db *gorm.DB) LocationRepository {
	return &locationRepository{db: db}
}

func (r *locationRepository) Create(location *model.Location) error {
	return r.db.Create(location).Error
}

func (r *locationRepository) FindByID(id uint) (*model.Location, error) {
	var location model.Location
	err := r.db.First(&location, id).Error
	if err != nil {
		return nil, err
	}
	return &location, nil
}

func (r *locationRepository) FindByNormalizedName(normalizedName string) (*model.Location, error) {
	var location model.Location
	err := r.db.Where("normalized_name = ?", normalizedName).First(&location).Error
	if err != nil {
		return nil, err
	}
	return &location, nil
}

func (r *locationRepository) FindAll() ([]model.Location, error) {
	var locations []model.Location
	err := r.db.Order("name ASC").Find(&locations).Error
	return locations, err
}

func (r *locationRepository) Update(location *model.Location) error {
	return r.db.Save(location).Error
}

func (r *locationRepository) Delete(id uint) error {
	return r.db.Delete(&model.Location{}, id).Error
}

// Merge reassigns every transaction pointing at sourceID to targetID, then soft-deletes
// the now-empty source location. Used by the management UI to collapse duplicates.
func (r *locationRepository) Merge(sourceID, targetID uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.Transaction{}).
			Where("location_id = ?", sourceID).
			Update("location_id", targetID).Error; err != nil {
			return err
		}
		return tx.Delete(&model.Location{}, sourceID).Error
	})
}
