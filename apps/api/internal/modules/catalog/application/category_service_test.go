package application_test

import (
	"context"
	"io"
	"testing"

	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/catalog/application"
	"github.com/adotomov/fashion-store/apps/api/internal/modules/catalog/domain"
)

type fakeMediaStorage struct{}

func (fakeMediaStorage) EnsureBucket(context.Context, string) error { return nil }
func (fakeMediaStorage) Upload(context.Context, string, string, string, io.Reader) (int64, error) {
	return 0, nil
}
func (fakeMediaStorage) Open(context.Context, string, string) (io.ReadCloser, string, error) {
	return nil, "", nil
}
func (fakeMediaStorage) Delete(context.Context, string, string) error { return nil }

type fakeCategoryRepo struct {
	byID      map[uuid.UUID]domain.Category
	slugsUsed map[string]bool
}

func newFakeCategoryRepo() *fakeCategoryRepo {
	return &fakeCategoryRepo{byID: map[uuid.UUID]domain.Category{}, slugsUsed: map[string]bool{}}
}

func (f *fakeCategoryRepo) Create(_ context.Context, category domain.Category) (*domain.Category, error) {
	if f.slugsUsed[category.Slug] {
		return nil, application.ErrSlugConflict
	}
	category.ID = uuid.New()
	f.slugsUsed[category.Slug] = true
	f.byID[category.ID] = category
	return &category, nil
}

func (f *fakeCategoryRepo) List(_ context.Context) ([]domain.Category, error) {
	var out []domain.Category
	for _, c := range f.byID {
		out = append(out, c)
	}
	return out, nil
}

func (f *fakeCategoryRepo) FindByID(_ context.Context, id uuid.UUID) (*domain.Category, error) {
	c, ok := f.byID[id]
	if !ok {
		return nil, domain.ErrCategoryNotFound
	}
	return &c, nil
}

func (f *fakeCategoryRepo) Update(_ context.Context, category domain.Category) (*domain.Category, error) {
	if _, ok := f.byID[category.ID]; !ok {
		return nil, domain.ErrCategoryNotFound
	}
	f.byID[category.ID] = category
	return &category, nil
}

func (f *fakeCategoryRepo) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := f.byID[id]; !ok {
		return domain.ErrCategoryNotFound
	}
	delete(f.byID, id)
	return nil
}

func TestCreateCategory_SupportsParent(t *testing.T) {
	svc := application.NewCategoryService(newFakeCategoryRepo(), fakeMediaStorage{}, "category-media")
	productTypeID := uuid.New()

	parent, err := svc.CreateCategory(context.Background(), application.CreateCategoryInput{Name: "Clothing", ProductTypeID: productTypeID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	child, err := svc.CreateCategory(context.Background(), application.CreateCategoryInput{
		Name:          "Dresses",
		ParentID:      &parent.ID,
		ProductTypeID: productTypeID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if child.ParentID == nil || *child.ParentID != parent.ID {
		t.Errorf("expected child category to reference parent %v, got %v", parent.ID, child.ParentID)
	}
}

func TestCreateCategory_RejectsEmptyName(t *testing.T) {
	svc := application.NewCategoryService(newFakeCategoryRepo(), fakeMediaStorage{}, "category-media")

	if _, err := svc.CreateCategory(context.Background(), application.CreateCategoryInput{Name: "", ProductTypeID: uuid.New()}); err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestCreateCategory_RejectsMissingProductType(t *testing.T) {
	svc := application.NewCategoryService(newFakeCategoryRepo(), fakeMediaStorage{}, "category-media")

	if _, err := svc.CreateCategory(context.Background(), application.CreateCategoryInput{Name: "Clothing"}); err == nil {
		t.Fatal("expected error for missing product_type_id")
	}
}
