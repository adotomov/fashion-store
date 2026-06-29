package application_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/catalog/application"
	"github.com/adotomov/fashion-store/apps/api/internal/modules/catalog/domain"
)

type fakeRepo struct {
	byID      map[uuid.UUID]domain.Catalog
	slugsUsed map[string]bool
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{byID: map[uuid.UUID]domain.Catalog{}, slugsUsed: map[string]bool{}}
}

func (f *fakeRepo) Create(_ context.Context, catalog domain.Catalog) (*domain.Catalog, error) {
	if f.slugsUsed[catalog.Slug] {
		return nil, application.ErrSlugConflict
	}
	catalog.ID = uuid.New()
	f.slugsUsed[catalog.Slug] = true
	f.byID[catalog.ID] = catalog
	return &catalog, nil
}

func (f *fakeRepo) List(_ context.Context) ([]domain.Catalog, error) {
	var out []domain.Catalog
	for _, c := range f.byID {
		out = append(out, c)
	}
	return out, nil
}

func (f *fakeRepo) FindByID(_ context.Context, id uuid.UUID) (*domain.Catalog, error) {
	c, ok := f.byID[id]
	if !ok {
		return nil, domain.ErrCatalogNotFound
	}
	return &c, nil
}

func (f *fakeRepo) Update(_ context.Context, catalog domain.Catalog) (*domain.Catalog, error) {
	if _, ok := f.byID[catalog.ID]; !ok {
		return nil, domain.ErrCatalogNotFound
	}
	f.byID[catalog.ID] = catalog
	return &catalog, nil
}

func (f *fakeRepo) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := f.byID[id]; !ok {
		return domain.ErrCatalogNotFound
	}
	delete(f.byID, id)
	return nil
}

func TestCreateCatalog_GeneratesSlugFromName(t *testing.T) {
	svc := application.NewCatalogService(newFakeRepo())

	catalog, err := svc.CreateCatalog(context.Background(), application.CreateCatalogInput{Name: "Summer Collection"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if catalog.Slug != "summer-collection" {
		t.Errorf("expected slug 'summer-collection', got %q", catalog.Slug)
	}
	if catalog.Status != domain.StatusDraft {
		t.Errorf("expected new catalog to default to draft status, got %q", catalog.Status)
	}
}

func TestCreateCatalog_RetriesOnSlugConflict(t *testing.T) {
	repo := newFakeRepo()
	svc := application.NewCatalogService(repo)

	if _, err := svc.CreateCatalog(context.Background(), application.CreateCatalogInput{Name: "Sale"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	second, err := svc.CreateCatalog(context.Background(), application.CreateCatalogInput{Name: "Sale"})
	if err != nil {
		t.Fatalf("expected slug conflict to be resolved by retry, got error: %v", err)
	}
	if second.Slug == "sale" {
		t.Error("expected second catalog to get a different slug than the first")
	}
}

func TestCreateCatalog_RejectsEmptyName(t *testing.T) {
	svc := application.NewCatalogService(newFakeRepo())

	if _, err := svc.CreateCatalog(context.Background(), application.CreateCatalogInput{Name: ""}); err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestUpdateCatalog_RejectsInvalidStatus(t *testing.T) {
	repo := newFakeRepo()
	svc := application.NewCatalogService(repo)

	catalog, err := svc.CreateCatalog(context.Background(), application.CreateCatalogInput{Name: "Winter"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	badStatus := domain.Status("archived")
	if _, err := svc.UpdateCatalog(context.Background(), catalog.ID, application.UpdateCatalogInput{Status: &badStatus}); err == nil {
		t.Fatal("expected error for invalid status")
	}
}
