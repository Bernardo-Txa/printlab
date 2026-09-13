package admin

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"
	"log"
	"mime"
	"path"
	"strconv"
	"strings"

	"github.com/Bernardo-Txa/printlab/internal/products"
)

const (
	AdminImageMaxBytes          int64 = 5 * 1024 * 1024
	adminImageRandomBytes             = 16
	adminImageAllowedTypesLabel       = "JPEG, PNG ou WebP"
)

var adminImageExtensions = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

func (s *Service) GetAdminProductImages(ctx context.Context, productID string) (AdminProductImagesPage, error) {
	if !s.catalogAvailable() {
		return AdminProductImagesPage{}, ErrUnavailable
	}
	productID = normalizeUUID(productID)
	if !ValidUUID(productID) {
		return AdminProductImagesPage{}, ErrInvalidCatalogID
	}

	page, err := s.catalog.GetAdminProductImagesPage(ctx, productID)
	if err != nil {
		return AdminProductImagesPage{}, err
	}
	s.prepareAdminProductImagesPage(&page)

	return page, nil
}

func (s *Service) AuthorizeAdminImageUpload(ctx context.Context, productID string, metadata AdminImageUploadMetadata) (AdminImageUploadAuthorization, error) {
	if !s.imageStorageAvailable() {
		return AdminImageUploadAuthorization{}, ErrStorageUnavailable
	}
	productID = normalizeUUID(productID)
	if !ValidUUID(productID) {
		return AdminImageUploadAuthorization{}, ErrInvalidCatalogID
	}

	contentType, _, err := validateAdminImageMetadata(metadata.ContentType, metadata.FileSize)
	if err != nil {
		return AdminImageUploadAuthorization{}, err
	}
	variantID, err := s.validateAdminImageVariant(ctx, productID, metadata.VariantID)
	if err != nil {
		return AdminImageUploadAuthorization{}, err
	}
	if _, err := s.catalog.GetAdminProductImagesPage(ctx, productID); err != nil {
		return AdminImageUploadAuthorization{}, err
	}

	objectPath, err := GenerateAdminImageObjectPath(productID, contentType, s.imageRandom)
	if err != nil {
		return AdminImageUploadAuthorization{}, ErrStorageUnavailable
	}
	_ = variantID

	signedUpload, err := s.storage.CreateSignedUpload(ctx, AdminImageBucket, objectPath)
	if err != nil {
		return AdminImageUploadAuthorization{}, ErrStorageUnavailable
	}

	return AdminImageUploadAuthorization{
		UploadURL:        signedUpload.UploadURL,
		ObjectPath:       objectPath,
		ContentType:      contentType,
		MaxFileSizeBytes: AdminImageMaxBytes,
	}, nil
}

func (s *Service) FinalizeAdminImageUpload(ctx context.Context, productID string, input AdminImageFinalizeInput) (string, error) {
	if !s.imageStorageAvailable() {
		return "", ErrStorageUnavailable
	}
	productID = normalizeUUID(productID)
	if !ValidUUID(productID) {
		return "", ErrInvalidCatalogID
	}

	createInput, err := s.validateAdminImageFinalizeInput(ctx, productID, input)
	if err != nil {
		return "", err
	}
	id, err := s.catalog.CreateAdminProductImage(ctx, createInput)
	if err != nil {
		if ManagedAdminImagePath(productID, createInput.StoragePath) {
			if cleanupErr := s.storage.DeleteObject(ctx, AdminImageBucket, createInput.StoragePath); cleanupErr != nil {
				log.Print("admin image cleanup after insert failure failed")
			}
		}
		return "", err
	}

	return id, nil
}

func (s *Service) AuthorizeAdminImageReplacement(ctx context.Context, productID string, imageID string, metadata AdminImageUploadMetadata) (AdminImageUploadAuthorization, error) {
	if !s.imageStorageAvailable() {
		return AdminImageUploadAuthorization{}, ErrStorageUnavailable
	}
	imageID = normalizeUUID(imageID)
	if !ValidUUID(imageID) {
		return AdminImageUploadAuthorization{}, ErrInvalidCatalogID
	}
	if err := s.ensureAdminProductImage(ctx, normalizeUUID(productID), imageID); err != nil {
		return AdminImageUploadAuthorization{}, err
	}

	return s.AuthorizeAdminImageUpload(ctx, productID, metadata)
}

func (s *Service) FinalizeAdminImageReplacement(ctx context.Context, productID string, imageID string, input AdminImageFinalizeInput) error {
	if !s.imageStorageAvailable() {
		return ErrStorageUnavailable
	}
	productID = normalizeUUID(productID)
	imageID = normalizeUUID(imageID)
	if !ValidUUID(productID) || !ValidUUID(imageID) {
		return ErrInvalidCatalogID
	}

	createInput, err := s.validateAdminImageFinalizeInput(ctx, productID, input)
	if err != nil {
		return err
	}
	oldImage, err := s.catalog.ReplaceAdminProductImage(ctx, AdminImageReplaceInput{
		ID:          imageID,
		ProductID:   productID,
		VariantID:   createInput.VariantID,
		StoragePath: createInput.StoragePath,
		AltText:     createInput.AltText,
		SortOrder:   createInput.SortOrder,
		IsPrimary:   createInput.IsPrimary,
	})
	if err != nil {
		if ManagedAdminImagePath(productID, createInput.StoragePath) {
			if cleanupErr := s.storage.DeleteObject(ctx, AdminImageBucket, createInput.StoragePath); cleanupErr != nil {
				log.Print("admin image cleanup after replacement failure failed")
			}
		}
		return err
	}
	if oldImage.StoragePath != createInput.StoragePath && ManagedAdminImagePath(productID, oldImage.StoragePath) {
		if err := s.storage.DeleteObject(ctx, AdminImageBucket, oldImage.StoragePath); err != nil {
			log.Print("admin image old object cleanup failed")
		}
	}

	return nil
}

func (s *Service) RemoveAdminProductImage(ctx context.Context, productID string, imageID string) error {
	if !s.catalogAvailable() {
		return ErrUnavailable
	}
	productID = normalizeUUID(productID)
	imageID = normalizeUUID(imageID)
	if !ValidUUID(productID) || !ValidUUID(imageID) {
		return ErrInvalidCatalogID
	}

	currentImage, err := s.getAdminProductImage(ctx, productID, imageID)
	if err != nil {
		return err
	}
	if s.storage == nil && ManagedAdminImagePath(productID, currentImage.StoragePath) {
		return ErrStorageUnavailable
	}

	image, err := s.catalog.DeleteAdminProductImage(ctx, productID, imageID)
	if err != nil {
		return err
	}
	if s.storage != nil && ManagedAdminImagePath(productID, image.StoragePath) {
		if err := s.storage.DeleteObject(ctx, AdminImageBucket, image.StoragePath); err != nil {
			log.Print("admin image storage delete failed")
		}
	}

	return nil
}

func (s *Service) UpdateAdminProductImageOrder(ctx context.Context, productID string, imageID string, sortOrder string) error {
	if !s.catalogAvailable() {
		return ErrUnavailable
	}
	productID = normalizeUUID(productID)
	imageID = normalizeUUID(imageID)
	if !ValidUUID(productID) || !ValidUUID(imageID) {
		return ErrInvalidCatalogID
	}
	order, err := parseAdminNonNegativeInt(sortOrder)
	if err != nil {
		return ErrValidation
	}

	return s.catalog.UpdateAdminProductImageOrder(ctx, AdminImageOrderInput{
		ID:        imageID,
		ProductID: productID,
		SortOrder: order,
	})
}

func (s *Service) MarkAdminProductImagePrimary(ctx context.Context, productID string, imageID string) error {
	if !s.catalogAvailable() {
		return ErrUnavailable
	}
	productID = normalizeUUID(productID)
	imageID = normalizeUUID(imageID)
	if !ValidUUID(productID) || !ValidUUID(imageID) {
		return ErrInvalidCatalogID
	}

	return s.catalog.MarkAdminProductImagePrimary(ctx, productID, imageID)
}

func (s *Service) validateAdminImageFinalizeInput(ctx context.Context, productID string, input AdminImageFinalizeInput) (AdminImageCreateInput, error) {
	contentType, _, err := validateAdminImageMetadata(input.ContentType, input.FileSize)
	if err != nil {
		return AdminImageCreateInput{}, err
	}
	objectPath := strings.TrimSpace(input.ObjectPath)
	if !ManagedAdminImagePath(productID, objectPath) {
		return AdminImageCreateInput{}, ErrInvalidImagePath
	}
	if input.SortOrder < 0 {
		return AdminImageCreateInput{}, ErrValidation
	}
	variantID, err := s.validateAdminImageVariant(ctx, productID, input.VariantID)
	if err != nil {
		return AdminImageCreateInput{}, err
	}

	object, err := s.storage.StatObject(ctx, AdminImageBucket, objectPath)
	if err != nil {
		return AdminImageCreateInput{}, storageError(err)
	}
	objectContentType, _, ok := AdminImageContentType(object.ContentType)
	if !ok || objectContentType != contentType {
		return AdminImageCreateInput{}, ErrInvalidImageType
	}
	if object.Size <= 0 || object.Size > AdminImageMaxBytes || object.Size != input.FileSize {
		return AdminImageCreateInput{}, ErrImageTooLarge
	}

	return AdminImageCreateInput{
		ProductID:   productID,
		VariantID:   variantID,
		StoragePath: objectPath,
		AltText:     strings.TrimSpace(input.AltText),
		SortOrder:   input.SortOrder,
		IsPrimary:   input.IsPrimary,
	}, nil
}

func (s *Service) validateAdminImageVariant(ctx context.Context, productID string, variantID string) (string, error) {
	variantID = normalizeUUID(variantID)
	if variantID == "" {
		return "", nil
	}
	if !ValidUUID(variantID) {
		return "", ErrInvalidCatalogID
	}
	if err := s.catalog.EnsureAdminImageVariant(ctx, productID, variantID); err != nil {
		return "", err
	}

	return variantID, nil
}

func (s *Service) ensureAdminProductImage(ctx context.Context, productID string, imageID string) error {
	productID = normalizeUUID(productID)
	imageID = normalizeUUID(imageID)
	if !ValidUUID(productID) || !ValidUUID(imageID) {
		return ErrInvalidCatalogID
	}
	page, err := s.catalog.GetAdminProductImagesPage(ctx, productID)
	if err != nil {
		return err
	}
	for _, image := range page.Images {
		if normalizeUUID(image.ID) == imageID {
			return nil
		}
	}

	return ErrCatalogNotFound
}

func (s *Service) getAdminProductImage(ctx context.Context, productID string, imageID string) (AdminProductImage, error) {
	productID = normalizeUUID(productID)
	imageID = normalizeUUID(imageID)
	if !ValidUUID(productID) || !ValidUUID(imageID) {
		return AdminProductImage{}, ErrInvalidCatalogID
	}
	page, err := s.catalog.GetAdminProductImagesPage(ctx, productID)
	if err != nil {
		return AdminProductImage{}, err
	}
	for _, image := range page.Images {
		if normalizeUUID(image.ID) == imageID {
			return image, nil
		}
	}

	return AdminProductImage{}, ErrCatalogNotFound
}

func (s *Service) prepareAdminProductImagesPage(page *AdminProductImagesPage) {
	if page == nil {
		return
	}
	page.StorageConfigured = s != nil && s.storage != nil
	page.UploadAction = "/admin/produtos/" + page.Product.ID + "/imagens/upload-url"
	page.MaxFileSizeBytes = AdminImageMaxBytes
	page.MaxFileSizeLabel = FormatAdminFileSize(AdminImageMaxBytes)
	page.AllowedContentTypes = adminImageAllowedTypesLabel

	for i := range page.Images {
		image := &page.Images[i]
		image.URL = products.PublicProductImageURL(s.supabaseURL, image.StoragePath)
		image.SortOrderInput = strconv.Itoa(image.SortOrder)
		image.ManagedStorageObject = ManagedAdminImagePath(page.Product.ID, image.StoragePath)
		image.ReplaceUploadAction = "/admin/produtos/" + page.Product.ID + "/imagens/" + image.ID + "/substituir-url"
		image.ReplaceFinalizeAction = "/admin/produtos/" + page.Product.ID + "/imagens/" + image.ID + "/finalizar-substituicao"
		image.RemoveAction = "/admin/produtos/" + page.Product.ID + "/imagens/" + image.ID + "/remover"
		image.OrderAction = "/admin/produtos/" + page.Product.ID + "/imagens/" + image.ID + "/ordem"
		image.PrimaryAction = "/admin/produtos/" + page.Product.ID + "/imagens/" + image.ID + "/principal"
		if image.VariantID == "" {
			image.ScopeLabel = "Produto"
			if image.AssociationLabel == "" {
				image.AssociationLabel = page.Product.Name
			}
		} else {
			image.ScopeLabel = "Configuração"
		}
	}
}

func validateAdminImageMetadata(contentType string, fileSize int64) (string, string, error) {
	canonicalContentType, extension, ok := AdminImageContentType(contentType)
	if !ok {
		return "", "", ErrInvalidImageType
	}
	if fileSize <= 0 {
		return "", "", ErrValidation
	}
	if fileSize > AdminImageMaxBytes {
		return "", "", ErrImageTooLarge
	}

	return canonicalContentType, extension, nil
}

func AdminImageContentType(value string) (string, string, bool) {
	mediaType, _, err := mime.ParseMediaType(strings.TrimSpace(value))
	if err != nil {
		mediaType = strings.TrimSpace(value)
	}
	mediaType = strings.ToLower(mediaType)
	extension, ok := adminImageExtensions[mediaType]

	return mediaType, extension, ok
}

func GenerateAdminImageObjectPath(productID string, contentType string, randomReader io.Reader) (string, error) {
	productID = normalizeUUID(productID)
	if !ValidUUID(productID) {
		return "", ErrInvalidCatalogID
	}
	_, extension, ok := AdminImageContentType(contentType)
	if !ok {
		return "", ErrInvalidImageType
	}
	if randomReader == nil {
		randomReader = rand.Reader
	}

	randomBytes := make([]byte, adminImageRandomBytes)
	if _, err := io.ReadFull(randomReader, randomBytes); err != nil {
		return "", err
	}

	return "products/" + productID + "/" + hex.EncodeToString(randomBytes) + extension, nil
}

func ManagedAdminImagePath(productID string, objectPath string) bool {
	productID = normalizeUUID(productID)
	objectPath = strings.TrimSpace(objectPath)
	if !ValidUUID(productID) || !validAdminStoragePath(objectPath) {
		return false
	}
	if !strings.HasPrefix(objectPath, "products/"+productID+"/") {
		return false
	}
	extension := strings.ToLower(path.Ext(objectPath))
	for _, allowedExtension := range adminImageExtensions {
		if extension == allowedExtension {
			return true
		}
	}

	return false
}

func validAdminStoragePath(storagePath string) bool {
	pathValue := strings.TrimSpace(storagePath)
	if pathValue == "" || pathValue != storagePath {
		return false
	}
	lowerPath := strings.ToLower(pathValue)
	if strings.HasPrefix(pathValue, "/") || strings.HasPrefix(pathValue, "//") || strings.HasPrefix(lowerPath, "http:") || strings.HasPrefix(lowerPath, "https:") {
		return false
	}
	if strings.Contains(strings.Split(pathValue, "/")[0], ":") {
		return false
	}
	for _, segment := range strings.Split(pathValue, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return false
		}
	}

	return true
}

func FormatAdminFileSize(bytes int64) string {
	if bytes%(1024*1024) == 0 {
		return strconv.FormatInt(bytes/(1024*1024), 10) + " MB"
	}

	return strconv.FormatInt(bytes, 10) + " bytes"
}

func (s *Service) imageStorageAvailable() bool {
	return s.catalogAvailable() && s.storage != nil
}
