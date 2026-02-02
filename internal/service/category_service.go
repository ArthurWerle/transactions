package service

import (
	"context"

	"github.com/ArthurWerle/transactions/internal/model"
	"github.com/ArthurWerle/transactions/internal/repository"
)

type CategoryService interface {
	CreateCategory(ctx context.Context, category *model.Category) error
	GetCategoryByID(ctx context.Context, id uint) (*model.Category, error)
	GetCategories(ctx context.Context) ([]model.Category, error)
	UpdateCategory(ctx context.Context, category *model.Category) error
	DeleteCategory(ctx context.Context, id uint) error
}

type categoryService struct {
	categoryRepo repository.CategoryRepository
}

func NewCategoryService(categoryRepo repository.CategoryRepository) CategoryService {
	return &categoryService{
		categoryRepo: categoryRepo,
	}
}

func (s *categoryService) CreateCategory(ctx context.Context, category *model.Category) error {
	return s.categoryRepo.Create(category)
}

func (s *categoryService) GetCategoryByID(ctx context.Context, id uint) (*model.Category, error) {
	return s.categoryRepo.FindByID(id)
}

func (s *categoryService) GetCategories(ctx context.Context) ([]model.Category, error) {
	return s.categoryRepo.FindAll()
}

func (s *categoryService) UpdateCategory(ctx context.Context, category *model.Category) error {
	return s.categoryRepo.Update(category)
}

func (s *categoryService) DeleteCategory(ctx context.Context, id uint) error {
	return s.categoryRepo.Delete(id)
}
