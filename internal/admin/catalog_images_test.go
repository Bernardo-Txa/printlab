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

func TestAuthorizeAdminImageReplacementValidatesImageOwnershipAndCreatesSignedURL(t *testing.T) {
	repo := newImageTestRepository()
	storage := &fakeStorageClient{signedUpload: SignedUpload{UploadURL: "https://storage.test/upload?token=signed"}}
	service := newImageTestService(repo, storage)

	authorization, err := service.AuthorizeAdminImageReplacement(context.Background(), repo.productID, repo.imageID, AdminImageUploadMetadata{
		ContentType: "image/webp",
		FileSize:    100,
	})
	if err != nil {
		t.Fatalf("expected replacement authorization, got %v", err)
	}
	if authorization.ObjectPath == "" || storage.createPath != authorization.ObjectPath {
		t.Fatalf("expected server-generated replacement path, got %#v", authorization)
	}

	storage.createPath = ""
	_, err = service.AuthorizeAdminImageReplacement(context.Background(), repo.productID, "44444444-4444-4444-4444-444444444444", AdminImageUploadMetadata{
		ContentType: "image/webp",
		FileSize:    100,
	})
	if !errors.Is(err, ErrCatalogNotFound) {
		t.Fatalf("expected ErrCatalogNotFound for image outside product, got %v", err)
	}
	if storage.createPath != "" {
		t.Fatal("expected unauthorized replacement not to create signed upload")
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

func TestFinalizeAdminImageReplacementUpdatesImageAndDeletesOldManagedObject(t *testing.T) {
	repo := newImageTestRepository()
	oldPath := "products/" + repo.productID + "/old.jpg"
	newPath := "products/" + repo.productID + "/new.webp"
	repo.replaceOldImage = AdminProductImage{StoragePath: oldPath}
	storage := &fakeStorageClient{object: StorageObject{ContentType: "image/webp", Size: 100}}
	service := newImageTestService(repo, storage)

	err := service.FinalizeAdminImageReplacement(context.Background(), repo.productID, repo.imageID, AdminImageFinalizeInput{
		ObjectPath:  newPath,
		ContentType: "image/webp",
		FileSize:    100,
		AltText:     "Nova foto",
		SortOrder:   4,
		IsPrimary:   true,
	})
	if err != nil {
		t.Fatalf("expected replacement success, got %v", err)
	}
	if !repo.replaceImageCalled || repo.replacedImage.StoragePath != newPath || repo.replacedImage.AltText != "Nova foto" {
		t.Fatalf("expected new object to be persisted, got %#v", repo.replacedImage)
	}
	if len(storage.deletedPaths) != 1 || storage.deletedPaths[0] != oldPath {
		t.Fatalf("expected old managed object cleanup after update, got %#v", storage.deletedPaths)
	}
}

func TestFinalizeAdminImageReplacementUpdateFailureCleansNewObjectAndKeepsOldObject(t *testing.T) {
	repo := newImageTestRepository()
	oldPath := "products/" + repo.productID + "/old.jpg"
	newPath := "products/" + repo.productID + "/new.png"
	repo.replaceOldImage = AdminProductImage{StoragePath: oldPath}
	repo.replaceImageErr = ErrUnavailable
	storage := &fakeStorageClient{object: StorageObject{ContentType: "image/png", Size: 100}}
	service := newImageTestService(repo, storage)

	err := service.FinalizeAdminImageReplacement(context.Background(), repo.productID, repo.imageID, AdminImageFinalizeInput{
		ObjectPath:  newPath,
		ContentType: "image/png",
		FileSize:    100,
	})
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("expected update failure, got %v", err)
	}
	if len(storage.deletedPaths) != 1 || storage.deletedPaths[0] != newPath {
		t.Fatalf("expected cleanup of only new object, got %#v", storage.deletedPaths)
	}
}

func TestFinalizeAdminImageReplacementDoesNotDeleteLegacyOldObject(t *testing.T) {
	repo := newImageTestRepository()
	newPath := "products/" + repo.productID + "/new.jpg"
	repo.replaceOldImage = AdminProductImage{StoragePath: "legacy/manual-image.jpg"}
	storage := &fakeStorageClient{object: StorageObject{ContentType: "image/jpeg", Size: 100}}
	service := newImageTestService(repo, storage)

	err := service.FinalizeAdminImageReplacement(context.Background(), repo.productID, repo.imageID, AdminImageFinalizeInput{
		ObjectPath:  newPath,
		ContentType: "image/jpeg",
		FileSize:    100,
	})
	if err != nil {
		t.Fatalf("expected replacement success, got %v", err)
	}
	if len(storage.deletedPaths) != 0 {
		t.Fatalf("expected legacy old object not to be deleted, got %#v", storage.deletedPaths)
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
	repo.page.Images = []AdminProductImage{{ID: repo.imageID, StoragePath: repo.deletedImage.StoragePath}}
	storage := &fakeStorageClient{}
	service := newImageTestService(repo, storage)

	if err := service.RemoveAdminProductImage(context.Background(), productID, "33333333-3333-3333-3333-333333333333"); err != nil {
		t.Fatalf("expected managed remove success, got %v", err)
	}
	if storage.deletedPath != repo.deletedImage.StoragePath || !repo.deleteImageCalled {
		t.Fatalf("expected managed object delete and DB delete, got path=%q delete=%v", storage.deletedPath, repo.deleteImageCalled)
	}

	repo.deletedImage = AdminProductImage{StoragePath: "legacy/manual-image.jpg"}
	repo.page.Images = []AdminProductImage{{ID: repo.imageID, StoragePath: repo.deletedImage.StoragePath}}
	repo.deleteImageCalled = false
	storage.deletedPath = ""
	storage.deletedPaths = nil
	if err := service.RemoveAdminProductImage(context.Background(), productID, "33333333-3333-3333-3333-333333333333"); err != nil {
		t.Fatalf("expected legacy remove success, got %v", err)
	}
	if storage.deletedPath != "" || !repo.deleteImageCalled {
		t.Fatalf("expected legacy association delete without storage delete, got path=%q delete=%v", storage.deletedPath, repo.deleteImageCalled)
	}
}

func TestRemoveAdminProductImageRequiresStorageForManagedObject(t *testing.T) {
	repo := newImageTestRepository()
	service := newImageTestService(repo, nil)

	err := service.RemoveAdminProductImage(context.Background(), repo.productID, repo.imageID)
	if !errors.Is(err, ErrStorageUnavailable) {
		t.Fatalf("expected ErrStorageUnavailable, got %v", err)
	}
	if repo.deleteImageCalled {
		t.Fatal("expected managed image association to remain when storage is unavailable")
	}
}

func TestUpdateAdminProductImageOrder(t *testing.T) {
	repo := newImageTestRepository()
	service := newImageTestService(repo, &fakeStorageClient{})

	err := service.UpdateAdminProductImageOrder(context.Background(), repo.productID, repo.imageID, "7")
	if err != nil {
		t.Fatalf("expected order update success, got %v", err)
	}
	if !repo.updateImageOrderCalled || repo.imageOrderInput.ProductID != repo.productID || repo.imageOrderInput.ID != repo.imageID || repo.imageOrderInput.SortOrder != 7 {
		t.Fatalf("unexpected order input: %#v", repo.imageOrderInput)
	}

	err = service.UpdateAdminProductImageOrder(context.Background(), repo.productID, repo.imageID, "-1")
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected ErrValidation for negative order, got %v", err)
	}
}

func TestMarkAdminProductImagePrimary(t *testing.T) {
	repo := newImageTestRepository()
	service := newImageTestService(repo, &fakeStorageClient{})

	err := service.MarkAdminProductImagePrimary(context.Background(), repo.productID, repo.imageID)
	if err != nil {
		t.Fatalf("expected mark primary success, got %v", err)
	}
	if !repo.markImagePrimaryCalled || repo.markProductID != repo.productID || repo.markImageID != repo.imageID {
		t.Fatalf("unexpected mark primary call: product=%q image=%q called=%v", repo.markProductID, repo.markImageID, repo.markImagePrimaryCalled)
	}
}

func newImageTestService(repo *imageTestRepository, storage *fakeStorageClient) *Service {
	options := []ServiceOption{
		WithAdminSupabaseURL("https://example.supabase.co"),
		WithAdminImageRandom(bytes.NewReader(bytes.Repeat([]byte{0x44}, adminImageRandomBytes))),
	}
	if storage != nil {
		options = append(options, WithStorageClient(storage))
	}

	return NewService(
		fakeAuthClient{userID: testAdminUserID},
		repo,
		testAdminUserID,
		CookieOptions{},
		options...,
	)
}

type fakeStorageClient struct {
	signedUpload SignedUpload
	object       StorageObject
	err          error
	createPath   string
	statCalled   bool
	deletedPath  string
	deletedPaths []string
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
	c.deletedPaths = append(c.deletedPaths, objectPath)
	return c.err
}

type imageTestRepository struct {
	*fakeRepository
	productID              string
	imageID                string
	page                   AdminProductImagesPage
	allowedVariantID       string
	ensureVariantCalled    bool
	ensureVariantErr       error
	createImageCalled      bool
	createdImage           AdminImageCreateInput
	createImageErr         error
	replaceImageCalled     bool
	replacedImage          AdminImageReplaceInput
	replaceOldImage        AdminProductImage
	replaceImageErr        error
	deleteImageCalled      bool
	deletedImage           AdminProductImage
	updateImageOrderCalled bool
	imageOrderInput        AdminImageOrderInput
	markImagePrimaryCalled bool
	markProductID          string
	markImageID            string
}

func newImageTestRepository() *imageTestRepository {
	productID := "11111111-1111-1111-1111-111111111111"
	imageID := "33333333-3333-3333-3333-333333333333"
	image := AdminProductImage{ID: imageID, ProductID: productID, StoragePath: "products/" + productID + "/old.jpg"}
	return &imageTestRepository{
		fakeRepository: newFakeRepository(),
		productID:      productID,
		imageID:        imageID,
		page: AdminProductImagesPage{
			Product: AdminProductHeader{
				ID:        productID,
				Name:      "Produto",
				Slug:      "produto",
				DetailURL: "/admin/produtos/" + productID,
			},
			Images: []AdminProductImage{image},
		},
		replaceOldImage: image,
		deletedImage:    image,
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
func (r *imageTestRepository) ReplaceAdminProductImage(_ context.Context, input AdminImageReplaceInput) (AdminProductImage, error) {
	r.replaceImageCalled = true
	r.replacedImage = input
	if r.replaceImageErr != nil {
		return r.replaceOldImage, r.replaceImageErr
	}
	return r.replaceOldImage, nil
}
func (r *imageTestRepository) DeleteAdminProductImage(context.Context, string, string) (AdminProductImage, error) {
	r.deleteImageCalled = true
	return r.deletedImage, nil
}
func (r *imageTestRepository) UpdateAdminProductImageOrder(_ context.Context, input AdminImageOrderInput) error {
	r.updateImageOrderCalled = true
	r.imageOrderInput = input
	return nil
}
func (r *imageTestRepository) MarkAdminProductImagePrimary(_ context.Context, productID string, imageID string) error {
	r.markImagePrimaryCalled = true
	r.markProductID = productID
	r.markImageID = imageID
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

func (r *imageTestRepository) ListAdminProductColors(context.Context, string) ([]AdminProductColorOption, error) {
	return nil, nil
}
