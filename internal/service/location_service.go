package service

import (
	"context"
	"errors"
	"strings"

	"github.com/ArthurWerle/transactions/internal/model"
	"github.com/ArthurWerle/transactions/internal/repository"
	"gorm.io/gorm"
)

type LocationService interface {
	FindOrCreate(ctx context.Context, name string) (*model.Location, error)
	GetLocationByID(ctx context.Context, id uint) (*model.Location, error)
	GetLocations(ctx context.Context) ([]model.Location, error)
	UpdateLocation(ctx context.Context, location *model.Location) error
	DeleteLocation(ctx context.Context, id uint) error
	MergeLocations(ctx context.Context, sourceID, targetID uint) error
}

type locationService struct {
	locationRepo repository.LocationRepository
}

func NewLocationService(locationRepo repository.LocationRepository) LocationService {
	return &locationService{
		locationRepo: locationRepo,
	}
}

// normalizeLocationName collapses casing and whitespace so that "Mercado X",
// "mercado x " and "MERCADO  X" all resolve to the same dedup key.
func normalizeLocationName(name string) string {
	return strings.ToLower(strings.Join(strings.Fields(name), " "))
}

// FindOrCreate returns the existing location matching the normalized name, or
// creates a new one keeping the caller's original casing as the display name.
func (s *locationService) FindOrCreate(ctx context.Context, name string) (*model.Location, error) {
	normalized := normalizeLocationName(name)
	if normalized == "" {
		return nil, errors.New("location name cannot be empty")
	}

	existing, err := s.locationRepo.FindByNormalizedName(normalized)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	location := &model.Location{
		Name:           strings.TrimSpace(name),
		NormalizedName: normalized,
	}
	if err := s.locationRepo.Create(location); err != nil {
		return nil, err
	}
	return location, nil
}

func (s *locationService) GetLocationByID(ctx context.Context, id uint) (*model.Location, error) {
	return s.locationRepo.FindByID(id)
}

func (s *locationService) GetLocations(ctx context.Context) ([]model.Location, error) {
	return s.locationRepo.FindAll()
}

func (s *locationService) UpdateLocation(ctx context.Context, location *model.Location) error {
	location.NormalizedName = normalizeLocationName(location.Name)
	return s.locationRepo.Update(location)
}

func (s *locationService) DeleteLocation(ctx context.Context, id uint) error {
	return s.locationRepo.Delete(id)
}

func (s *locationService) MergeLocations(ctx context.Context, sourceID, targetID uint) error {
	if sourceID == targetID {
		return errors.New("source and target locations must be different")
	}
	return s.locationRepo.Merge(sourceID, targetID)
}
