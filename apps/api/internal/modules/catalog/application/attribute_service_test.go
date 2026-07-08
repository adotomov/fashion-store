package application_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/catalog/application"
	"github.com/adotomov/fashion-store/apps/api/internal/modules/catalog/domain"
)

type fakeAttributeRepo struct {
	byID   map[uuid.UUID]domain.Attribute
	values map[uuid.UUID][]domain.AttributeValue
}

func newFakeAttributeRepo() *fakeAttributeRepo {
	return &fakeAttributeRepo{byID: map[uuid.UUID]domain.Attribute{}, values: map[uuid.UUID][]domain.AttributeValue{}}
}

func (f *fakeAttributeRepo) Create(_ context.Context, attribute domain.Attribute) (*domain.Attribute, error) {
	attribute.ID = uuid.New()
	f.byID[attribute.ID] = attribute
	return &attribute, nil
}

func (f *fakeAttributeRepo) List(_ context.Context) ([]domain.Attribute, error) {
	var out []domain.Attribute
	for _, a := range f.byID {
		a.Values = f.values[a.ID]
		out = append(out, a)
	}
	return out, nil
}

func (f *fakeAttributeRepo) FindByID(_ context.Context, id uuid.UUID) (*domain.Attribute, error) {
	a, ok := f.byID[id]
	if !ok {
		return nil, domain.ErrAttributeNotFound
	}
	a.Values = f.values[id]
	return &a, nil
}

func (f *fakeAttributeRepo) UpdateName(_ context.Context, id uuid.UUID, name string) (*domain.Attribute, error) {
	a, ok := f.byID[id]
	if !ok {
		return nil, domain.ErrAttributeNotFound
	}
	a.Name = name
	f.byID[id] = a
	a.Values = f.values[id]
	return &a, nil
}

func (f *fakeAttributeRepo) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := f.byID[id]; !ok {
		return domain.ErrAttributeNotFound
	}
	delete(f.byID, id)
	return nil
}

func (f *fakeAttributeRepo) AddValue(_ context.Context, attributeID uuid.UUID, value string, colorHex *string) (*domain.AttributeValue, error) {
	if _, ok := f.byID[attributeID]; !ok {
		return nil, domain.ErrAttributeNotFound
	}
	v := domain.AttributeValue{ID: uuid.New(), AttributeID: attributeID, Value: value, ColorHex: colorHex}
	f.values[attributeID] = append(f.values[attributeID], v)
	return &v, nil
}

func (f *fakeAttributeRepo) DeleteValue(_ context.Context, attributeID, valueID uuid.UUID) error {
	values := f.values[attributeID]
	for i, v := range values {
		if v.ID == valueID {
			f.values[attributeID] = append(values[:i], values[i+1:]...)
			return nil
		}
	}
	return domain.ErrAttributeValueNotFound
}

func TestAttributeService_CreateAndAddValues(t *testing.T) {
	svc := application.NewAttributeService(newFakeAttributeRepo())

	attribute, err := svc.CreateAttribute(context.Background(), application.CreateAttributeInput{Name: "Size"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := svc.AddValue(context.Background(), attribute.ID, "S", nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := svc.AddValue(context.Background(), attribute.ID, "M", nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated, err := svc.GetAttribute(context.Background(), attribute.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(updated.Values) != 2 {
		t.Errorf("expected 2 values, got %d", len(updated.Values))
	}
}

func TestAttributeService_RejectsEmptyName(t *testing.T) {
	svc := application.NewAttributeService(newFakeAttributeRepo())

	if _, err := svc.CreateAttribute(context.Background(), application.CreateAttributeInput{Name: ""}); err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestAttributeService_ColorValueRequiresValidHex(t *testing.T) {
	repo := newFakeAttributeRepo()
	svc := application.NewAttributeService(repo)

	color, err := repo.Create(context.Background(), domain.Attribute{Name: "Color", Type: domain.AttributeTypeColor, IsSystem: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Missing hex on a color attribute is rejected.
	if _, err := svc.AddValue(context.Background(), color.ID, "Clay", nil); err == nil {
		t.Fatal("expected error when color hex is missing")
	}
	// Malformed hex is rejected.
	bad := "clay"
	if _, err := svc.AddValue(context.Background(), color.ID, "Clay", &bad); err == nil {
		t.Fatal("expected error for malformed hex")
	}
	// Valid hex succeeds and is persisted on the value.
	good := "#B2543C"
	v, err := svc.AddValue(context.Background(), color.ID, "Clay", &good)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.ColorHex == nil || *v.ColorHex != good {
		t.Errorf("expected color hex %q, got %v", good, v.ColorHex)
	}
}

func TestAttributeService_DeleteSystemAttributeBlocked(t *testing.T) {
	repo := newFakeAttributeRepo()
	svc := application.NewAttributeService(repo)

	color, err := repo.Create(context.Background(), domain.Attribute{Name: "Color", Type: domain.AttributeTypeColor, IsSystem: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := svc.DeleteAttribute(context.Background(), color.ID); err != domain.ErrSystemAttributeReadOnly {
		t.Fatalf("expected ErrSystemAttributeReadOnly, got %v", err)
	}
}
