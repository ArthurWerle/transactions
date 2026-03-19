package repository

import (
	"github.com/ArthurWerle/transactions/internal/model"
	"gorm.io/gorm"
)

type SubcategoryRepository interface {
	Create(subcategory *model.Subcategory) error
	FindByID(id uint) (*model.Subcategory, error)
	FindAll() ([]model.Subcategory, error)
	Update(subcategory *model.Subcategory) error
	Delete(id uint) error
}

type subcategoryRepository struct {
	db *gorm.DB
}

func NewSubcategoryRepository(db *gorm.DB) SubcategoryRepository {
	return &subcategoryRepository{db: db}
}

func (r *subcategoryRepository) Create(subcategory *model.Subcategory) error {
	return r.db.Create(subcategory).Error
}

func (r *subcategoryRepository) FindByID(id uint) (*model.Subcategory, error) {
	var subcategory model.Subcategory
	err := r.db.First(&subcategory, id).Error
	if err != nil {
		return nil, err
	}
	return &subcategory, nil
}

func (r *subcategoryRepository) FindAll() ([]model.Subcategory, error) {
	var subcategories []model.Subcategory
	err := r.db.Order("name ASC").Find(&subcategories).Error
	return subcategories, err
}

func (r *subcategoryRepository) Update(subcategory *model.Subcategory) error {
	return r.db.Save(subcategory).Error
}

func (r *subcategoryRepository) Delete(id uint) error {
	return r.db.Delete(&model.Subcategory{}, id).Error
}
