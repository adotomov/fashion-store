package infrastructure_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"

	"github.com/adotomov/fashion-store/apps/api/internal/modules/catalog/application"
	"github.com/adotomov/fashion-store/apps/api/internal/modules/catalog/domain"
	"github.com/adotomov/fashion-store/apps/api/internal/modules/catalog/infrastructure"
	"github.com/adotomov/fashion-store/apps/api/internal/platform/storage"
)

// Full real-stack test: real Postgres + real FakeGCS, exercised through
// ProductService exactly as the HTTP handler would call it. Skips if
// DATABASE_URL isn't set or FakeGCS isn't reachable.
func TestProductService_UploadServeDeleteMedia_RealStorage(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()

	endpoint := os.Getenv("STORAGE_ENDPOINT")
	if endpoint == "" {
		endpoint = "https://localhost:4443"
	}
	client := storage.NewClient(endpoint, true)

	if err := client.EnsureBucket(ctx, "it-test-media"); err != nil {
		t.Skipf("FakeGCS not reachable, skipping: %v", err)
	}

	productRepo := infrastructure.NewPostgresProductRepository(pool)
	service := application.NewProductService(productRepo, client, "it-test-media")

	product, err := productRepo.Create(ctx, domain.Product{
		Name:   "IT-Test Media Product",
		Slug:   "it-test-media-product",
		Status: domain.ProductStatusDraft,
	})
	if err != nil {
		t.Fatalf("create product: %v", err)
	}
	t.Cleanup(func() { _ = productRepo.Delete(ctx, product.ID) })

	content := []byte("fake image bytes")
	media, err := service.UploadMedia(ctx, product.ID, "photo.jpg", "image/jpeg", bytes.NewReader(content), 0, "A test photo")
	if err != nil {
		t.Fatalf("upload media: %v", err)
	}
	if media.SizeBytes != int64(len(content)) {
		t.Errorf("expected size %d, got %d", len(content), media.SizeBytes)
	}

	reader, contentType, err := service.OpenMedia(ctx, media.ID)
	if err != nil {
		t.Fatalf("open media: %v", err)
	}
	got, err := io.ReadAll(reader)
	reader.Close()
	if err != nil {
		t.Fatalf("read media: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("expected content %q, got %q", content, got)
	}
	if contentType != "image/jpeg" {
		t.Errorf("expected content type image/jpeg, got %q", contentType)
	}

	if err := service.DeleteMedia(ctx, media.ID); err != nil {
		t.Fatalf("delete media: %v", err)
	}

	if _, err := productRepo.FindMediaByID(ctx, media.ID); err == nil {
		t.Error("expected media to be deleted from DB")
	}
}
