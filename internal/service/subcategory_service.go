package service

import (
	"context"

	"github.com/ArthurWerle/transactions/internal/model"
	"github.com/ArthurWerle/transactions/internal/repository"
)

type SubcategoryService interface {
	CreateSubcategory(ctx context.Context, subcategory *model.Subcategory) error
	GetSubcategoryByID(ctx context.Context, id uint) (*model.Subcategory, error)
	GetSubcategories(ctx context.Context) ([]model.Subcategory, error)
	UpdateSubcategory(ctx context.Context, subcategory *model.Subcategory) error
	DeleteSubcategory(ctx context.Context, id uint) error
}

type subcategoryService struct {
	subcategoryRepo repository.SubcategoryRepository
}

func NewSubcategoryService(subcategoryRepo repository.SubcategoryRepository) SubcategoryService {
	return &subcategoryService{
		subcategoryRepo: subcategoryRepo,
	}
}

func (s *subcategoryService) CreateSubcategory(ctx context.Context, subcategory *model.Subcategory) error {
	return s.subcategoryRepo.Create(subcategory)
}

func (s *subcategoryService) GetSubcategoryByID(ctx context.Context, id uint) (*model.Subcategory, error) {
	return s.subcategoryRepo.FindByID(id)
}

func (s *subcategoryService) GetSubcategories(ctx context.Context) ([]model.Subcategory, error) {
	return s.subcategoryRepo.FindAll()
}

func (s *subcategoryService) UpdateSubcategory(ctx context.Context, subcategory *model.Subcategory) error {
	return s.subcategoryRepo.Update(subcategory)
}

func (s *subcategoryService) DeleteSubcategory(ctx context.Context, id uint) error {
	return s.subcategoryRepo.Delete(id)
}
