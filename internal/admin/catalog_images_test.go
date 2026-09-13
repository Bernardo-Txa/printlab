package admin

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
)

func TestGenerateAdminImageObjectPathUsesServerRandomAndCanonicalExtension(t *testing.T) {
	productID := "11111111-1111-1111-1111-111111111111"
	first, err := GenerateAdminImageObjectPath(productID, "image/jpeg", bytes.NewReader(bytes.Repeat([]byte{0x11}, adminImageRandomBytes)))
	if err != nil {
		t.Fatalf("expected path, got %v", err)
	}
	second, err := GenerateAdminImageObjectPath(productID, "image/jpeg", bytes.NewReader(bytes.Repeat([]byte{0x22}, adminImageRandomBytes)))
	if err != nil {
		t.Fatalf("expected second path, got %v", err)
	}

	if first == second {
		t.Fatal("expected different random paths")
	}
	if !strings.HasPrefix(first, "products/"+productID+"/") || !strings.HasSuffix(first, ".jpg") {
		t.Fatalf("expected managed jpg path, got %q", first)
	}
	if strings.Contains(first, "original-name") {
		t.Fatalf("expected path not to contain client filename, got %q", first)
	}
}

func TestAdminImageContentTypeAndSizeValidation(t *testing.T) {
	if _, _, ok := AdminImageContentType("image/svg+xml"); ok {
		t.Fatal("expected SVG to be rejected")
	}
	if _, _, err := validateAdminImageMetadata("image/png", AdminImageMaxBytes+1); !errors.Is(err, ErrImageTooLarge) {
		t.Fatalf("expected ErrImageTooLarge, got %v", err)
	}
	if _, _, err := validateAdminImageMetadata("image/webp", AdminImageMaxBytes); err != nil {
		t.Fatalf("expected max bucket size to be accepted, got %v", err)
	}
}

func TestAuthorizeAdminImageUploadValidatesVariantAndCreatesSignedURL(t *testing.T) {
	repo := newImageTestRepository()
	repo.allowedVariantID = "22222222-2222-2222-2222-222222222222"
	storage := &fakeStorageClient{signedUpload: SignedUpload{UploadURL: "https://storage.test/upload?token=signed"}}
	service := newImageTestService(repo, storage)

	authorization, err := service.AuthorizeAdminImageUpload(context.Background(), "11111111-1111-1111-1111-111111111111", AdminImageUploadMetadata{
		VariantID:   repo.allowedVariantID,
		ContentType: "image/png",
		FileSize:    100,
		Filename:    "bad.svg",
	})
	if err != nil {
		t.Fatalf("expected authorization, got %v", err)
	}
	if !repo.ensureVariantCalled || storage.createPath == "" {
		t.Fatal("expected variant validation and storage authorization")
	}
	if authorization.ObjectPath != storage.createPath {
		t.Fatalf("expected object path from storage request, got %#v", authorization)
	}
	if strings.Contains(authorization.ObjectPath, "bad.svg") || !strings.HasSuffix(authorization.ObjectPath, ".png") {
		t.Fatalf("expected server-generated png path, got %q", authorization.ObjectPath)
	}
}

func TestAuthorizeAdminImageUploadRejectsVariantFromAnotherProduct(t *testing.T) {
	repo := newImageTestRepository()
	repo.ensureVariantErr = ErrCatalogNotFound
	service := newImageTestService(repo, &fakeStorageClient{})

	_, err := service.AuthorizeAdminImageUpload(context.Background(), "11111111-1111-1111-1111-111111111111", AdminImageUploadMetadata{
		VariantID:   "22222222-2222-2222-2222-222222222222",
		ContentType: "image/png",
		FileSize:    100,
	})
	if !errors.Is(err, ErrCatalogNotFound) {
		t.Fatalf("expected ErrCatalogNotFound, got %v", err)
	}
}

func TestFinalizeAdminImageUploadStatsObjectAndCreatesProductImage(t *testing.T) {
	repo := newImageTestRepository()
	storage := &fakeStorageClient{object: StorageObject{ContentType: "image/webp", Size: 100}}
	service := newImageTestService(repo, storage)
	objectPath := "products/11111111-1111-1111-1111-111111111111/random.webp"

	id, err := service.FinalizeAdminImageUpload(context.Background(), "11111111-1111-1111-1111-111111111111", AdminImageFinalizeInput{
		ObjectPath:  objectPath,
		ContentType: "image/webp",
		FileSize:    100,
		AltText:     "Foto do produto",
		SortOrder:   3,
		IsPrimary:   true,
		Filename:    "../ignored.png",
	})
	if err != nil {
		t.Fatalf("expected finalize success, got %v", err)
	}
	if id == "" || !storage.statCalled || !repo.createImageCalled {
		t.Fatalf("expected stat and insert, id=%q stat=%v create=%v", id, storage.statCalled, repo.createImageCalled)
	}
	if repo.createdImage.StoragePath != objectPath || repo.createdImage.AltText != "Foto do produto" || !repo.createdImage.IsPrimary {
		t.Fatalf("unexpected created image input: %#v", repo.createdImage)
	}
}

func TestFinalizeAdminImageUploadCleansUpObjectWhenInsertFails(t *testing.T) {
	repo := newImageTestRepository()
	repo.createImageErr = ErrUnavailable
	storage := &fakeStorageClient{object: StorageObject{ContentType: "image/jpeg", Size: 100}}
	service := newImageTestService(repo, storage)
	objectPath := "products/11111111-1111-1111-1111-111111111111/random.jpg"

	_, err := service.FinalizeAdminImageUpload(context.Background(), "11111111-1111-1111-1111-111111111111", AdminImageFinalizeInput{
		ObjectPath:  objectPath,
		ContentType: "image/jpeg",
		FileSize:    100,
	})
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("expected insert error, got %v", err)
	}
	if storage.deletedPath != objectPath {
		t.Fatalf("expected cleanup delete for %q, got %q", objectPath, storage.deletedPath)
	}
}

func TestFinalizeAdminImageUploadRejectsInvalidObjectMetadata(t *testing.T) {
	repo := newImageTestRepository()
	storage := &fakeStorageClient{object: StorageObject{ContentType: "image/png", Size: 100}}
	service := newImageTestService(repo, storage)

	_, err := service.FinalizeAdminImageUpload(context.Background(), "11111111-1111-1111-1111-111111111111", AdminImageFinalizeInput{
		ObjectPath:  "products/11111111-1111-1111-1111-111111111111/random.png",
		ContentType: "image/webp",
		FileSize:    100,
	})
	if !errors.Is(err, ErrInvalidImageType) {
		t.Fatalf("expected ErrInvalidImageType, got %v", err)
	}
	if repo.createImageCalled {
		t.Fatal("expected invalid object not to create product image")
	}
}

func TestRemoveAdminProductImageDeletesOnlyManagedStorageObject(t *testing.T) {
	productID := "11111111-1111-1111-1111-111111111111"
	repo := newImageTestRepository()
	repo.deletedImage = AdminProductImage{StoragePath: "products/" + productID + "/random.jpg"}
	storage := &fakeStorageClient{}
	service := newImageTestService(repo, storage)

	if err := service.RemoveAdminProductImage(context.Background(), productID, "33333333-3333-3333-3333-333333333333"); err != nil {
		t.Fatalf("expected managed remove success, got %v", err)
	}
	if storage.deletedPath != repo.deletedImage.StoragePath {
		t.Fatalf("expected managed object delete, got %q", storage.deletedPath)
	}

	repo.deletedImage = AdminProductImage{StoragePath: "legacy/manual-image.jpg"}
	storage.deletedPath = ""
	if err := service.RemoveAdminProductImage(context.Background(), productID, "33333333-3333-3333-3333-333333333333"); err != nil {
		t.Fatalf("expected legacy remove success, got %v", err)
	}
	if storage.deletedPath != "" {
		t.Fatalf("expected legacy image not to delete storage object, got %q", storage.deletedPath)
	}
}

func newImageTestService(repo *imageTestRepository, storage *fakeStorageClient) *Service {
	return NewService(
		fakeAuthClient{userID: testAdminUserID},
		repo,
		testAdminUserID,
		CookieOptions{},
		WithStorageClient(storage),
		WithAdminSupabaseURL("https://example.supabase.co"),
		WithAdminImageRandom(bytes.NewReader(bytes.Repeat([]byte{0x44}, adminImageRandomBytes))),
	)
}

type fakeStorageClient struct {
	signedUpload SignedUpload
	object       StorageObject
	err          error
	createPath   string
	statCalled   bool
	deletedPath  string
}

func (c *fakeStorageClient) CreateSignedUpload(_ context.Context, _ string, objectPath string) (SignedUpload, error) {
	if c.err != nil {
		return SignedUpload{}, c.err
	}
	c.createPath = objectPath
	c.signedUpload.Path = objectPath
	if c.signedUpload.UploadURL == "" {
		c.signedUpload.UploadURL = "https://storage.test/upload?token=signed"
	}
	return c.signedUpload, nil
}

func (c *fakeStorageClient) StatObject(context.Context, string, string) (StorageObject, error) {
	c.statCalled = true
	if c.err != nil {
		return StorageObject{}, c.err
	}
	return c.object, nil
}

func (c *fakeStorageClient) DeleteObject(_ context.Context, _ string, objectPath string) error {
	c.deletedPath = objectPath
	return c.err
}

type imageTestRepository struct {
	*fakeRepository
	page                AdminProductImagesPage
	allowedVariantID    string
	ensureVariantCalled bool
	ensureVariantErr    error
	createImageCalled   bool
	createdImage        AdminImageCreateInput
	createImageErr      error
	deletedImage        AdminProductImage
}

func newImageTestRepository() *imageTestRepository {
	productID := "11111111-1111-1111-1111-111111111111"
	return &imageTestRepository{
		fakeRepository: newFakeRepository(),
		page: AdminProductImagesPage{
			Product: AdminProductHeader{
				ID:        productID,
				Name:      "Produto",
				Slug:      "produto",
				DetailURL: "/admin/produtos/" + productID,
			},
		},
		deletedImage: AdminProductImage{StoragePath: "products/" + productID + "/random.jpg"},
	}
}

func (r *imageTestRepository) ListAdminProducts(context.Context, AdminProductListFilter) (AdminProductListPage, error) {
	return AdminProductListPage{}, nil
}
func (r *imageTestRepository) GetAdminProductForm(context.Context, string) (AdminProductFormPage, error) {
	return AdminProductFormPage{}, nil
}
func (r *imageTestRepository) CreateAdminProduct(context.Context, AdminProductSaveInput) (string, error) {
	return "", nil
}
func (r *imageTestRepository) UpdateAdminProduct(context.Context, AdminProductSaveInput) error {
	return nil
}
func (r *imageTestRepository) GetAdminProductImagesPage(context.Context, string) (AdminProductImagesPage, error) {
	return r.page, nil
}
func (r *imageTestRepository) EnsureAdminImageVariant(_ context.Context, _ string, variantID string) error {
	r.ensureVariantCalled = true
	if r.ensureVariantErr != nil {
		return r.ensureVariantErr
	}
	if r.allowedVariantID != "" && variantID != r.allowedVariantID {
		return ErrCatalogNotFound
	}
	return nil
}
func (r *imageTestRepository) CreateAdminProductImage(_ context.Context, input AdminImageCreateInput) (string, error) {
	r.createImageCalled = true
	r.createdImage = input
	if r.createImageErr != nil {
		return "", r.createImageErr
	}
	return "33333333-3333-3333-3333-333333333333", nil
}
func (r *imageTestRepository) ReplaceAdminProductImage(context.Context, AdminImageReplaceInput) (AdminProductImage, error) {
	return AdminProductImage{}, nil
}
func (r *imageTestRepository) DeleteAdminProductImage(context.Context, string, string) (AdminProductImage, error) {
	return r.deletedImage, nil
}
func (r *imageTestRepository) UpdateAdminProductImageOrder(context.Context, AdminImageOrderInput) error {
	return nil
}
func (r *imageTestRepository) MarkAdminProductImagePrimary(context.Context, string, string) error {
	return nil
}
func (r *imageTestRepository) ListAdminCategories(context.Context) (AdminCategoryListPage, error) {
	return AdminCategoryListPage{}, nil
}
func (r *imageTestRepository) GetAdminCategoryForm(context.Context, string) (AdminCategoryFormPage, error) {
	return AdminCategoryFormPage{}, nil
}
func (r *imageTestRepository) CreateAdminCategory(context.Context, AdminCategorySaveInput) (string, error) {
	return "", nil
}
func (r *imageTestRepository) UpdateAdminCategory(context.Context, AdminCategorySaveInput) error {
	return nil
}
func (r *imageTestRepository) GetAdminVariantForm(context.Context, string, string) (AdminVariantFormPage, error) {
	return AdminVariantFormPage{}, nil
}
func (r *imageTestRepository) NewAdminVariantForm(context.Context, string) (AdminVariantFormPage, error) {
	return AdminVariantFormPage{}, nil
}
func (r *imageTestRepository) CreateAdminVariant(context.Context, AdminVariantSaveInput) (string, error) {
	return "", nil
}
func (r *imageTestRepository) UpdateAdminVariant(context.Context, AdminVariantSaveInput) error {
	return nil
}
func (r *imageTestRepository) AddAdminRecipeComponent(context.Context, AdminRecipeSaveInput) error {
	return nil
}
func (r *imageTestRepository) UpdateAdminRecipeComponent(context.Context, AdminRecipeSaveInput) error {
	return nil
}
func (r *imageTestRepository) RemoveAdminRecipeComponent(context.Context, string, string, string) error {
	return nil
}
func (r *imageTestRepository) ListAdminMaterials(context.Context) (AdminMaterialListPage, error) {
	return AdminMaterialListPage{}, nil
}
func (r *imageTestRepository) GetAdminMaterialForm(context.Context, string) (AdminMaterialFormPage, error) {
	return AdminMaterialFormPage{}, nil
}
func (r *imageTestRepository) CreateAdminMaterial(context.Context, AdminMaterialSaveInput) (string, error) {
	return "", nil
}
func (r *imageTestRepository) UpdateAdminMaterial(context.Context, AdminMaterialSaveInput) error {
	return nil
}
func (r *imageTestRepository) ListAdminColors(context.Context) (AdminColorListPage, error) {
	return AdminColorListPage{}, nil
}
func (r *imageTestRepository) GetAdminColorForm(context.Context, string) (AdminColorFormPage, error) {
	return AdminColorFormPage{}, nil
}
func (r *imageTestRepository) CreateAdminColor(context.Context, AdminColorSaveInput) (string, error) {
	return "", nil
}
func (r *imageTestRepository) UpdateAdminColor(context.Context, AdminColorSaveInput) error {
	return nil
}
func (r *imageTestRepository) ListAdminBoxes(context.Context) (AdminBoxListPage, error) {
	return AdminBoxListPage{}, nil
}
func (r *imageTestRepository) GetAdminBoxForm(context.Context, string) (AdminBoxFormPage, error) {
	return AdminBoxFormPage{}, nil
}
func (r *imageTestRepository) CreateAdminBox(context.Context, AdminBoxSaveInput) (string, error) {
	return "", nil
}
func (r *imageTestRepository) UpdateAdminBox(context.Context, AdminBoxSaveInput) error {
	return nil
}
