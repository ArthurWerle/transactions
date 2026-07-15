package service

import (
	"context"
	"testing"

	"github.com/ArthurWerle/transactions/internal/model"
	"gorm.io/gorm"
)

type mockLocationRepository struct {
	locations    map[uint]*model.Location
	mergedSource uint
	mergedTarget uint
}

func newMockLocationRepository() *mockLocationRepository {
	return &mockLocationRepository{locations: make(map[uint]*model.Location)}
}

func (m *mockLocationRepository) Create(location *model.Location) error {
	if location.ID == 0 {
		location.ID = uint(len(m.locations) + 1)
	}
	m.locations[location.ID] = location
	return nil
}

func (m *mockLocationRepository) FindByID(id uint) (*model.Location, error) {
	l, ok := m.locations[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return l, nil
}

func (m *mockLocationRepository) FindByNormalizedName(normalizedName string) (*model.Location, error) {
	for _, l := range m.locations {
		if l.NormalizedName == normalizedName {
			return l, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockLocationRepository) FindAll() ([]model.Location, error) {
	var out []model.Location
	for _, l := range m.locations {
		out = append(out, *l)
	}
	return out, nil
}

func (m *mockLocationRepository) Update(location *model.Location) error {
	m.locations[location.ID] = location
	return nil
}

func (m *mockLocationRepository) Delete(id uint) error {
	delete(m.locations, id)
	return nil
}

func (m *mockLocationRepository) Merge(sourceID, targetID uint) error {
	m.mergedSource = sourceID
	m.mergedTarget = targetID
	delete(m.locations, sourceID)
	return nil
}

func TestFindOrCreate_DeduplicatesByNormalizedName(t *testing.T) {
	repo := newMockLocationRepository()
	svc := NewLocationService(repo)
	ctx := context.Background()

	first, err := svc.FindOrCreate(ctx, "Mercado X")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Casing and extra whitespace must resolve to the same record.
	for _, variant := range []string{"mercado x ", "MERCADO  X", "  Mercado X  "} {
		got, err := svc.FindOrCreate(ctx, variant)
		if err != nil {
			t.Fatalf("unexpected error for %q: %v", variant, err)
		}
		if got.ID != first.ID {
			t.Errorf("variant %q created a new location (id=%d), expected reuse of id=%d", variant, got.ID, first.ID)
		}
	}

	if len(repo.locations) != 1 {
		t.Errorf("expected 1 location stored, got %d", len(repo.locations))
	}

	// The display name keeps the original casing of the first insertion.
	if first.Name != "Mercado X" {
		t.Errorf("expected display name 'Mercado X', got %q", first.Name)
	}
}

func TestFindOrCreate_CreatesDistinctNames(t *testing.T) {
	repo := newMockLocationRepository()
	svc := NewLocationService(repo)
	ctx := context.Background()

	a, _ := svc.FindOrCreate(ctx, "Mercado X")
	b, _ := svc.FindOrCreate(ctx, "Lancheria Y")

	if a.ID == b.ID {
		t.Errorf("distinct names should produce distinct locations, both got id=%d", a.ID)
	}
	if len(repo.locations) != 2 {
		t.Errorf("expected 2 locations stored, got %d", len(repo.locations))
	}
}

// raceyLocationRepository simulates a concurrent request winning the insert
// between our lookup and our create: the first FindByNormalizedName misses,
// then Create fails with the unique-index error, and subsequent lookups find
// the row the "other" request inserted.
type raceyLocationRepository struct {
	*mockLocationRepository
	winner *model.Location
}

func (r *raceyLocationRepository) FindByNormalizedName(normalizedName string) (*model.Location, error) {
	return r.mockLocationRepository.FindByNormalizedName(normalizedName)
}

func (r *raceyLocationRepository) Create(location *model.Location) error {
	// The concurrent winner lands right before our insert.
	r.winner = &model.Location{Name: location.Name, NormalizedName: location.NormalizedName}
	r.winner.ID = 42
	r.mockLocationRepository.locations[r.winner.ID] = r.winner
	return gorm.ErrDuplicatedKey
}

func TestFindOrCreate_RecoversFromDuplicateKeyRace(t *testing.T) {
	repo := &raceyLocationRepository{mockLocationRepository: newMockLocationRepository()}
	svc := NewLocationService(repo)

	got, err := svc.FindOrCreate(context.Background(), "SUPERMERCADO BROMBATTI")
	if err != nil {
		t.Fatalf("expected duplicate-key race to be recovered, got error: %v", err)
	}
	if got.ID != repo.winner.ID {
		t.Errorf("expected the concurrently created location (id=%d), got id=%d", repo.winner.ID, got.ID)
	}
}

func TestFindOrCreate_RejectsEmptyName(t *testing.T) {
	svc := NewLocationService(newMockLocationRepository())
	if _, err := svc.FindOrCreate(context.Background(), "   "); err == nil {
		t.Error("expected error for blank location name, got nil")
	}
}

func TestMergeLocations(t *testing.T) {
	repo := newMockLocationRepository()
	svc := NewLocationService(repo)
	ctx := context.Background()

	source, _ := svc.FindOrCreate(ctx, "mercdo x")
	target, _ := svc.FindOrCreate(ctx, "Mercado X")

	if err := svc.MergeLocations(ctx, source.ID, target.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.mergedSource != source.ID || repo.mergedTarget != target.ID {
		t.Errorf("merge called with (%d,%d), expected (%d,%d)", repo.mergedSource, repo.mergedTarget, source.ID, target.ID)
	}
	if _, ok := repo.locations[source.ID]; ok {
		t.Error("source location should have been removed after merge")
	}
}

func TestMergeLocations_RejectsSameSourceAndTarget(t *testing.T) {
	svc := NewLocationService(newMockLocationRepository())
	if err := svc.MergeLocations(context.Background(), 1, 1); err == nil {
		t.Error("expected error when merging a location into itself, got nil")
	}
}
