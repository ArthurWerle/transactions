package repository

import (
	"github.com/ArthurWerle/transactions/internal/model"
	"gorm.io/gorm"
)

type CategoryRepository interface {
	Create(category *model.Category) error
	FindByID(id uint) (*model.Category, error)
	FindAll(includeDeleted bool) ([]model.Category, error)
	Update(category *model.Category) error
	Delete(id uint) error
}

type categoryRepository struct {
	db *gorm.DB
}

func NewCategoryRepository(db *gorm.DB) CategoryRepository {
	return &categoryRepository{db: db}
}

func (r *categoryRepository) Create(category *model.Category) error {
	return r.db.Create(category).Error
}

func (r *categoryRepository) FindByID(id uint) (*model.Category, error) {
	var category model.Category
	err := r.db.First(&category, id).Error
	if err != nil {
		return nil, err
	}
	return &category, nil
}

// FindAll returns live categories; with includeDeleted it also returns
// soft-deleted ones (their deleted_at populated), so reporting consumers can
// label money that belongs to removed categories instead of hiding it.
func (r *categoryRepository) FindAll(includeDeleted bool) ([]model.Category, error) {
	var categories []model.Category
	query := r.db
	if includeDeleted {
		query = query.Unscoped()
	}
	err := query.Order("name ASC").Find(&categories).Error
	return categories, err
}

func (r *categoryRepository) Update(category *model.Category) error {
	return r.db.Save(category).Error
}

func (r *categoryRepository) Delete(id uint) error {
	return r.db.Delete(&model.Category{}, id).Error
}
