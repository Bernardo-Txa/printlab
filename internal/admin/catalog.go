package admin

import (
	"context"
	"errors"
	"math"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/Bernardo-Txa/printlab/internal/products"
)

const (
	AdminCatalogStatusAll      = "all"
	AdminCatalogStatusActive   = "active"
	AdminCatalogStatusInactive = "inactive"
)

var hexColorPattern = regexp.MustCompile(`^#[0-9A-F]{6}$`)

type CatalogRepository interface {
	ListAdminProducts(ctx context.Context, filter AdminProductListFilter) (AdminProductListPage, error)
	GetAdminProductForm(ctx context.Context, productID string) (AdminProductFormPage, error)
	CreateAdminProduct(ctx context.Context, input AdminProductSaveInput) (string, error)
	UpdateAdminProduct(ctx context.Context, input AdminProductSaveInput) error
	ListAdminCategories(ctx context.Context) (AdminCategoryListPage, error)
	GetAdminCategoryForm(ctx context.Context, categoryID string) (AdminCategoryFormPage, error)
	CreateAdminCategory(ctx context.Context, input AdminCategorySaveInput) (string, error)
	UpdateAdminCategory(ctx context.Context, input AdminCategorySaveInput) error
	GetAdminVariantForm(ctx context.Context, productID string, variantID string) (AdminVariantFormPage, error)
	NewAdminVariantForm(ctx context.Context, productID string) (AdminVariantFormPage, error)
	CreateAdminVariant(ctx context.Context, input AdminVariantSaveInput) (string, error)
	UpdateAdminVariant(ctx context.Context, input AdminVariantSaveInput) error
	AddAdminRecipeComponent(ctx context.Context, input AdminRecipeSaveInput) error
	UpdateAdminRecipeComponent(ctx context.Context, input AdminRecipeSaveInput) error
	RemoveAdminRecipeComponent(ctx context.Context, productID string, variantID string, componentID string) error
	ListAdminMaterials(ctx context.Context) (AdminMaterialListPage, error)
	GetAdminMaterialForm(ctx context.Context, materialID string) (AdminMaterialFormPage, error)
	CreateAdminMaterial(ctx context.Context, input AdminMaterialSaveInput) (string, error)
	UpdateAdminMaterial(ctx context.Context, input AdminMaterialSaveInput) error
	ListAdminColors(ctx context.Context) (AdminColorListPage, error)
	GetAdminColorForm(ctx context.Context, colorID string) (AdminColorFormPage, error)
	CreateAdminColor(ctx context.Context, input AdminColorSaveInput) (string, error)
	UpdateAdminColor(ctx context.Context, input AdminColorSaveInput) error
	ListAdminBoxes(ctx context.Context) (AdminBoxListPage, error)
	GetAdminBoxForm(ctx context.Context, boxID string) (AdminBoxFormPage, error)
	CreateAdminBox(ctx context.Context, input AdminBoxSaveInput) (string, error)
	UpdateAdminBox(ctx context.Context, input AdminBoxSaveInput) error
}

func NormalizeAdminProductListFilter(filter AdminProductListFilter) AdminProductListFilter {
	filter.Status = strings.TrimSpace(filter.Status)
	if filter.Status != AdminCatalogStatusActive && filter.Status != AdminCatalogStatusInactive {
		filter.Status = AdminCatalogStatusAll
	}
	filter.Query = strings.TrimSpace(filter.Query)

	return filter
}

func AdminProductListOptions(activeStatus string, query string) []AdminFilterOption {
	filter := NormalizeAdminProductListFilter(AdminProductListFilter{Status: activeStatus, Query: query})
	options := []struct {
		value string
		label string
	}{
		{AdminCatalogStatusAll, "Todos"},
		{AdminCatalogStatusActive, "Ativos"},
		{AdminCatalogStatusInactive, "Inativos"},
	}

	result := make([]AdminFilterOption, 0, len(options))
	for _, option := range options {
		result = append(result, AdminFilterOption{
			Value:  option.value,
			Label:  option.label,
			URL:    AdminProductsURL(option.value, filter.Query),
			Active: option.value == filter.Status,
		})
	}

	return result
}

func AdminProductsURL(status string, query string) string {
	filter := NormalizeAdminProductListFilter(AdminProductListFilter{Status: status, Query: query})
	values := url.Values{}
	if filter.Status != AdminCatalogStatusAll {
		values.Set("status", filter.Status)
	}
	if filter.Query != "" {
		values.Set("q", filter.Query)
	}
	if encoded := values.Encode(); encoded != "" {
		return "/admin/produtos?" + encoded
	}

	return "/admin/produtos"
}

func PrepareAdminProductListPage(filter AdminProductListFilter, items []AdminProductListItem) AdminProductListPage {
	filter = NormalizeAdminProductListFilter(filter)
	for i := range items {
		PrepareAdminProductListItem(&items[i])
	}

	return AdminProductListPage{
		Products: items,
		Status:   filter.Status,
		Query:    filter.Query,
		Options:  AdminProductListOptions(filter.Status, filter.Query),
	}
}

func PrepareAdminProductListItem(item *AdminProductListItem) {
	if item == nil {
		return
	}

	item.PriceBRL = products.FormatBRL(item.PriceCents)
	item.StatusLabel = ActiveStatusLabel(item.IsActive)
	if item.IsFeatured {
		item.FeaturedLabel = "Destaque"
	} else {
		item.FeaturedLabel = "Sem destaque"
	}
	if item.CategoryName == "" {
		item.CategoryLabel = "Sem categoria"
	} else if item.CategoryActive {
		item.CategoryLabel = item.CategoryName
	} else {
		item.CategoryLabel = item.CategoryName + " — inativa"
	}
	item.VariantCountLabel = strconv.Itoa(item.VariantCount)
	if item.VariantCount == 1 {
		item.VariantCountLabel += " variante"
	} else {
		item.VariantCountLabel += " variantes"
	}
	if item.HasShippingProfile {
		item.ShippingLabel = "Perfil logistico presente"
	} else {
		item.ShippingLabel = "Sem perfil logistico"
		item.Warnings = append(item.Warnings, "Sem perfil logistico efetivo no produto.")
	}
	if item.VariantCount == 0 {
		item.Warnings = append(item.Warnings, "Sem variantes cadastradas.")
	}
	item.DetailURL = "/admin/produtos/" + strings.ToLower(item.ID)
	item.VariantsURL = item.DetailURL
}

func ActiveStatusLabel(active bool) string {
	if active {
		return "Ativo"
	}

	return "Inativo"
}

func FormatAdminInt(value int) string {
	return strconv.Itoa(value)
}

func FormatAdminInt64(value int64) string {
	return strconv.FormatInt(value, 10)
}

func FormatAdminCentsInput(cents int64) string {
	reais := cents / 100
	centavos := cents % 100

	return strconv.FormatInt(reais, 10) + "," + leftPadAdmin2(centavos)
}

func FormatAdminWeightInput(mg int64) string {
	grams := mg / 1000
	milligrams := mg % 1000
	if milligrams == 0 {
		return strconv.FormatInt(grams, 10)
	}

	fraction := strconv.FormatInt(milligrams+1000, 10)[1:]
	fraction = strings.TrimRight(fraction, "0")
	return strconv.FormatInt(grams, 10) + "," + fraction
}

func FormatAdminDimensionsLabel(height int, width int, length int) string {
	return strconv.Itoa(height) + " x " + strconv.Itoa(width) + " x " + strconv.Itoa(length) + " mm"
}

func ParseAdminBRLCents(value string) (int64, error) {
	return parseScaledAdminDecimal(value, 100, 2, true)
}

func ParseAdminGramsToMilligrams(value string) (int64, error) {
	mg, err := parseScaledAdminDecimal(value, 1000, 3, true)
	if err != nil || mg <= 0 {
		return 0, ErrValidation
	}

	return mg, nil
}

func CanonicalSlug(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	var builder strings.Builder
	lastDash := false
	for _, char := range value {
		mapped := canonicalSlugRune(char)
		if mapped == 0 {
			if builder.Len() > 0 && !lastDash {
				builder.WriteByte('-')
				lastDash = true
			}
			continue
		}

		builder.WriteByte(mapped)
		lastDash = false
	}

	return strings.Trim(builder.String(), "-")
}

func NormalizeHexColor(value string) (string, error) {
	value = strings.ToUpper(strings.TrimSpace(value))
	if value == "" {
		return "", nil
	}
	if !hexColorPattern.MatchString(value) {
		return "", ErrValidation
	}

	return value, nil
}

func (s *Service) ListAdminProducts(ctx context.Context, filter AdminProductListFilter) (AdminProductListPage, error) {
	if !s.catalogAvailable() {
		return AdminProductListPage{}, ErrUnavailable
	}

	return s.catalog.ListAdminProducts(ctx, NormalizeAdminProductListFilter(filter))
}

func (s *Service) NewAdminProduct(ctx context.Context) (AdminProductFormPage, error) {
	if !s.catalogAvailable() {
		return AdminProductFormPage{}, ErrUnavailable
	}

	page, err := s.catalog.GetAdminProductForm(ctx, "")
	if err != nil {
		return AdminProductFormPage{}, err
	}
	page.Title = "Novo produto"
	page.Action = "/admin/produtos"
	page.SubmitLabel = "Criar produto"
	page.BackURL = "/admin/produtos"
	page.IsNew = true
	page.Form.IsActive = true

	return page, nil
}

func (s *Service) GetAdminProduct(ctx context.Context, productID string) (AdminProductFormPage, error) {
	if !s.catalogAvailable() {
		return AdminProductFormPage{}, ErrUnavailable
	}
	productID = normalizeUUID(productID)
	if !ValidUUID(productID) {
		return AdminProductFormPage{}, ErrInvalidCatalogID
	}

	return s.catalog.GetAdminProductForm(ctx, productID)
}

func (s *Service) CreateAdminProduct(ctx context.Context, form AdminProductForm) (string, AdminProductFormPage, error) {
	page, err := s.NewAdminProduct(ctx)
	if err != nil {
		return "", AdminProductFormPage{}, err
	}
	page.Form = form
	input, errorsByField := validateAdminProductForm(form, true, "")
	if errorsByField.Any() {
		page.Errors = errorsByField
		return "", page, ErrValidation
	}

	id, err := s.catalog.CreateAdminProduct(ctx, input)
	if err != nil {
		page.Errors = catalogSaveErrors(err)
		return "", page, err
	}

	return id, page, nil
}

func (s *Service) UpdateAdminProduct(ctx context.Context, productID string, form AdminProductForm) (AdminProductFormPage, error) {
	page, err := s.GetAdminProduct(ctx, productID)
	if err != nil {
		return AdminProductFormPage{}, err
	}
	page.Form = form
	input, errorsByField := validateAdminProductForm(form, false, productID)
	if errorsByField.Any() {
		page.Errors = errorsByField
		return page, ErrValidation
	}

	input.ID = normalizeUUID(productID)
	if err := s.catalog.UpdateAdminProduct(ctx, input); err != nil {
		page.Errors = catalogSaveErrors(err)
		return page, err
	}

	return page, nil
}

func (s *Service) ListAdminCategories(ctx context.Context) (AdminCategoryListPage, error) {
	if !s.catalogAvailable() {
		return AdminCategoryListPage{}, ErrUnavailable
	}

	return s.catalog.ListAdminCategories(ctx)
}

func (s *Service) NewAdminCategory() AdminCategoryFormPage {
	return AdminCategoryFormPage{
		Title:       "Nova categoria",
		Action:      "/admin/categorias",
		SubmitLabel: "Criar categoria",
		BackURL:     "/admin/categorias",
		IsNew:       true,
		Form:        AdminCategoryForm{IsActive: true},
	}
}

func (s *Service) GetAdminCategory(ctx context.Context, categoryID string) (AdminCategoryFormPage, error) {
	if !s.catalogAvailable() {
		return AdminCategoryFormPage{}, ErrUnavailable
	}
	categoryID = normalizeUUID(categoryID)
	if !ValidUUID(categoryID) {
		return AdminCategoryFormPage{}, ErrInvalidCatalogID
	}

	return s.catalog.GetAdminCategoryForm(ctx, categoryID)
}

func (s *Service) CreateAdminCategory(ctx context.Context, form AdminCategoryForm) (string, AdminCategoryFormPage, error) {
	if !s.catalogAvailable() {
		return "", AdminCategoryFormPage{}, ErrUnavailable
	}
	page := s.NewAdminCategory()
	page.Form = form
	input, errorsByField := validateAdminCategoryForm(form, true, "")
	if errorsByField.Any() {
		page.Errors = errorsByField
		return "", page, ErrValidation
	}

	id, err := s.catalog.CreateAdminCategory(ctx, input)
	if err != nil {
		page.Errors = catalogSaveErrors(err)
		return "", page, err
	}

	return id, page, nil
}

func (s *Service) UpdateAdminCategory(ctx context.Context, categoryID string, form AdminCategoryForm) (AdminCategoryFormPage, error) {
	page, err := s.GetAdminCategory(ctx, categoryID)
	if err != nil {
		return AdminCategoryFormPage{}, err
	}
	page.Form = form
	input, errorsByField := validateAdminCategoryForm(form, false, categoryID)
	if errorsByField.Any() {
		page.Errors = errorsByField
		return page, ErrValidation
	}
	input.ID = normalizeUUID(categoryID)
	if err := s.catalog.UpdateAdminCategory(ctx, input); err != nil {
		page.Errors = catalogSaveErrors(err)
		return page, err
	}

	return page, nil
}

func (s *Service) NewAdminVariant(ctx context.Context, productID string) (AdminVariantFormPage, error) {
	if !s.catalogAvailable() {
		return AdminVariantFormPage{}, ErrUnavailable
	}
	productID = normalizeUUID(productID)
	if !ValidUUID(productID) {
		return AdminVariantFormPage{}, ErrInvalidCatalogID
	}

	page, err := s.catalog.NewAdminVariantForm(ctx, productID)
	if err != nil {
		return AdminVariantFormPage{}, err
	}
	page.Form.IsActive = true
	page.Form.SortOrder = "0"
	return page, nil
}

func (s *Service) GetAdminVariant(ctx context.Context, productID string, variantID string) (AdminVariantFormPage, error) {
	if !s.catalogAvailable() {
		return AdminVariantFormPage{}, ErrUnavailable
	}
	productID = normalizeUUID(productID)
	variantID = normalizeUUID(variantID)
	if !ValidUUID(productID) || !ValidUUID(variantID) {
		return AdminVariantFormPage{}, ErrInvalidCatalogID
	}

	return s.catalog.GetAdminVariantForm(ctx, productID, variantID)
}

func (s *Service) CreateAdminVariant(ctx context.Context, productID string, form AdminVariantForm) (string, AdminVariantFormPage, error) {
	page, err := s.NewAdminVariant(ctx, productID)
	if err != nil {
		return "", AdminVariantFormPage{}, err
	}
	page.Form = form
	input, errorsByField := validateAdminVariantForm(form, true, productID, "")
	if errorsByField.Any() {
		page.Errors = errorsByField
		return "", page, ErrValidation
	}

	id, err := s.catalog.CreateAdminVariant(ctx, input)
	if err != nil {
		page.Errors = catalogSaveErrors(err)
		return "", page, err
	}

	return id, page, nil
}

func (s *Service) UpdateAdminVariant(ctx context.Context, productID string, variantID string, form AdminVariantForm) (AdminVariantFormPage, error) {
	page, err := s.GetAdminVariant(ctx, productID, variantID)
	if err != nil {
		return AdminVariantFormPage{}, err
	}
	page.Form = form
	input, errorsByField := validateAdminVariantForm(form, false, productID, variantID)
	if errorsByField.Any() {
		page.Errors = errorsByField
		return page, ErrValidation
	}

	if err := s.catalog.UpdateAdminVariant(ctx, input); err != nil {
		page.Errors = catalogSaveErrors(err)
		return page, err
	}

	return page, nil
}

func (s *Service) AddAdminRecipeComponent(ctx context.Context, productID string, variantID string, form AdminRecipeForm) error {
	if !s.catalogAvailable() {
		return ErrUnavailable
	}
	input, fieldErrors := validateAdminRecipeForm(form, productID, variantID, "")
	if fieldErrors.Any() {
		return ErrValidation
	}

	return s.catalog.AddAdminRecipeComponent(ctx, input)
}

func (s *Service) UpdateAdminRecipeComponent(ctx context.Context, productID string, variantID string, componentID string, form AdminRecipeForm) error {
	if !s.catalogAvailable() {
		return ErrUnavailable
	}
	input, fieldErrors := validateAdminRecipeForm(form, productID, variantID, componentID)
	if fieldErrors.Any() {
		return ErrValidation
	}

	return s.catalog.UpdateAdminRecipeComponent(ctx, input)
}

func (s *Service) RemoveAdminRecipeComponent(ctx context.Context, productID string, variantID string, componentID string) error {
	if !s.catalogAvailable() {
		return ErrUnavailable
	}
	productID = normalizeUUID(productID)
	variantID = normalizeUUID(variantID)
	componentID = normalizeUUID(componentID)
	if !ValidUUID(productID) || !ValidUUID(variantID) || !ValidUUID(componentID) {
		return ErrInvalidCatalogID
	}

	return s.catalog.RemoveAdminRecipeComponent(ctx, productID, variantID, componentID)
}

func (s *Service) ListAdminMaterials(ctx context.Context) (AdminMaterialListPage, error) {
	if !s.catalogAvailable() {
		return AdminMaterialListPage{}, ErrUnavailable
	}

	return s.catalog.ListAdminMaterials(ctx)
}

func (s *Service) NewAdminMaterial() AdminMaterialFormPage {
	return AdminMaterialFormPage{
		Title:       "Novo material",
		Action:      "/admin/materiais",
		SubmitLabel: "Criar material",
		BackURL:     "/admin/materiais",
		IsNew:       true,
		Form:        AdminMaterialForm{IsActive: true},
	}
}

func (s *Service) GetAdminMaterial(ctx context.Context, materialID string) (AdminMaterialFormPage, error) {
	if !s.catalogAvailable() {
		return AdminMaterialFormPage{}, ErrUnavailable
	}
	materialID = normalizeUUID(materialID)
	if !ValidUUID(materialID) {
		return AdminMaterialFormPage{}, ErrInvalidCatalogID
	}

	return s.catalog.GetAdminMaterialForm(ctx, materialID)
}

func (s *Service) CreateAdminMaterial(ctx context.Context, form AdminMaterialForm) (string, AdminMaterialFormPage, error) {
	if !s.catalogAvailable() {
		return "", AdminMaterialFormPage{}, ErrUnavailable
	}
	page := s.NewAdminMaterial()
	page.Form = form
	input, errorsByField := validateAdminMaterialForm(form, true, "")
	if errorsByField.Any() {
		page.Errors = errorsByField
		return "", page, ErrValidation
	}

	id, err := s.catalog.CreateAdminMaterial(ctx, input)
	if err != nil {
		page.Errors = catalogSaveErrors(err)
		return "", page, err
	}

	return id, page, nil
}

func (s *Service) UpdateAdminMaterial(ctx context.Context, materialID string, form AdminMaterialForm) (AdminMaterialFormPage, error) {
	page, err := s.GetAdminMaterial(ctx, materialID)
	if err != nil {
		return AdminMaterialFormPage{}, err
	}
	page.Form = form
	input, errorsByField := validateAdminMaterialForm(form, false, materialID)
	if errorsByField.Any() {
		page.Errors = errorsByField
		return page, ErrValidation
	}
	input.ID = normalizeUUID(materialID)
	if err := s.catalog.UpdateAdminMaterial(ctx, input); err != nil {
		page.Errors = catalogSaveErrors(err)
		return page, err
	}

	return page, nil
}

func (s *Service) ListAdminColors(ctx context.Context) (AdminColorListPage, error) {
	if !s.catalogAvailable() {
		return AdminColorListPage{}, ErrUnavailable
	}

	return s.catalog.ListAdminColors(ctx)
}

func (s *Service) NewAdminColor() AdminColorFormPage {
	return AdminColorFormPage{
		Title:       "Nova cor",
		Action:      "/admin/cores",
		SubmitLabel: "Criar cor",
		BackURL:     "/admin/cores",
		IsNew:       true,
		Form:        AdminColorForm{IsActive: true},
	}
}

func (s *Service) GetAdminColor(ctx context.Context, colorID string) (AdminColorFormPage, error) {
	if !s.catalogAvailable() {
		return AdminColorFormPage{}, ErrUnavailable
	}
	colorID = normalizeUUID(colorID)
	if !ValidUUID(colorID) {
		return AdminColorFormPage{}, ErrInvalidCatalogID
	}

	return s.catalog.GetAdminColorForm(ctx, colorID)
}

func (s *Service) CreateAdminColor(ctx context.Context, form AdminColorForm) (string, AdminColorFormPage, error) {
	if !s.catalogAvailable() {
		return "", AdminColorFormPage{}, ErrUnavailable
	}
	page := s.NewAdminColor()
	page.Form = form
	input, errorsByField := validateAdminColorForm(form, true, "")
	if errorsByField.Any() {
		page.Errors = errorsByField
		return "", page, ErrValidation
	}

	id, err := s.catalog.CreateAdminColor(ctx, input)
	if err != nil {
		page.Errors = catalogSaveErrors(err)
		return "", page, err
	}

	return id, page, nil
}

func (s *Service) UpdateAdminColor(ctx context.Context, colorID string, form AdminColorForm) (AdminColorFormPage, error) {
	page, err := s.GetAdminColor(ctx, colorID)
	if err != nil {
		return AdminColorFormPage{}, err
	}
	page.Form = form
	input, errorsByField := validateAdminColorForm(form, false, colorID)
	if errorsByField.Any() {
		page.Errors = errorsByField
		return page, ErrValidation
	}
	input.ID = normalizeUUID(colorID)
	if err := s.catalog.UpdateAdminColor(ctx, input); err != nil {
		page.Errors = catalogSaveErrors(err)
		return page, err
	}

	return page, nil
}

func (s *Service) ListAdminBoxes(ctx context.Context) (AdminBoxListPage, error) {
	if !s.catalogAvailable() {
		return AdminBoxListPage{}, ErrUnavailable
	}

	return s.catalog.ListAdminBoxes(ctx)
}

func (s *Service) NewAdminBox() AdminBoxFormPage {
	return AdminBoxFormPage{
		Title:       "Nova caixa",
		Action:      "/admin/caixas",
		SubmitLabel: "Criar caixa",
		BackURL:     "/admin/caixas",
		IsNew:       true,
		Form: AdminBoxForm{
			IsActive:  true,
			SortOrder: "0",
		},
	}
}

func (s *Service) GetAdminBox(ctx context.Context, boxID string) (AdminBoxFormPage, error) {
	if !s.catalogAvailable() {
		return AdminBoxFormPage{}, ErrUnavailable
	}
	boxID = normalizeUUID(boxID)
	if !ValidUUID(boxID) {
		return AdminBoxFormPage{}, ErrInvalidCatalogID
	}

	return s.catalog.GetAdminBoxForm(ctx, boxID)
}

func (s *Service) CreateAdminBox(ctx context.Context, form AdminBoxForm) (string, AdminBoxFormPage, error) {
	if !s.catalogAvailable() {
		return "", AdminBoxFormPage{}, ErrUnavailable
	}
	page := s.NewAdminBox()
	page.Form = form
	input, errorsByField := validateAdminBoxForm(form, true, "")
	if errorsByField.Any() {
		page.Errors = errorsByField
		return "", page, ErrValidation
	}

	id, err := s.catalog.CreateAdminBox(ctx, input)
	if err != nil {
		page.Errors = catalogSaveErrors(err)
		return "", page, err
	}

	return id, page, nil
}

func (s *Service) UpdateAdminBox(ctx context.Context, boxID string, form AdminBoxForm) (AdminBoxFormPage, error) {
	page, err := s.GetAdminBox(ctx, boxID)
	if err != nil {
		return AdminBoxFormPage{}, err
	}
	page.Form = form
	input, errorsByField := validateAdminBoxForm(form, false, boxID)
	if errorsByField.Any() {
		page.Errors = errorsByField
		return page, ErrValidation
	}
	input.ID = normalizeUUID(boxID)
	if err := s.catalog.UpdateAdminBox(ctx, input); err != nil {
		page.Errors = catalogSaveErrors(err)
		return page, err
	}

	return page, nil
}

func (s *Service) catalogAvailable() bool {
	return s.Available() && s.catalog != nil
}

func validateAdminProductForm(form AdminProductForm, create bool, id string) (AdminProductSaveInput, AdminFieldErrors) {
	errorsByField := AdminFieldErrors{}
	input := AdminProductSaveInput{
		ID:               normalizeUUID(id),
		Name:             strings.TrimSpace(form.Name),
		Slug:             strings.TrimSpace(form.Slug),
		CategoryID:       normalizeUUID(form.CategoryID),
		ShortDescription: strings.TrimSpace(form.ShortDescription),
		Description:      strings.TrimSpace(form.Description),
		IsActive:         form.IsActive,
		IsFeatured:       form.IsFeatured,
	}
	if input.Name == "" {
		errorsByField.Add("name", "Informe o nome.")
	}
	if input.Slug == "" && create {
		input.Slug = CanonicalSlug(input.Name)
	}
	validateSlugField(errorsByField, "slug", input.Slug, "Informe um slug válido.")
	if form.CategoryID != "" && !ValidUUID(input.CategoryID) {
		errorsByField.Add("category_id", "Selecione uma categoria valida.")
	}
	price, err := ParseAdminBRLCents(form.PriceBRL)
	if err != nil {
		errorsByField.Add("price", "Informe um preco em reais, como 39,90.")
	}
	input.PriceCents = price
	input.ShippingProfile = validateAdminShippingProfile(errorsByField, form.UseShipping, form.ShippingWeightG, form.ShippingHeightMM, form.ShippingWidthMM, form.ShippingLengthMM)

	return input, errorsByField
}

func validateAdminCategoryForm(form AdminCategoryForm, create bool, id string) (AdminCategorySaveInput, AdminFieldErrors) {
	errorsByField := AdminFieldErrors{}
	input := AdminCategorySaveInput{
		ID:          normalizeUUID(id),
		Name:        strings.TrimSpace(form.Name),
		Slug:        strings.TrimSpace(form.Slug),
		Description: strings.TrimSpace(form.Description),
		IsActive:    form.IsActive,
	}
	if input.Name == "" {
		errorsByField.Add("name", "Informe o nome.")
	}
	if input.Slug == "" && create {
		input.Slug = CanonicalSlug(input.Name)
	}
	validateSlugField(errorsByField, "slug", input.Slug, "Informe um slug válido.")

	return input, errorsByField
}

func validateAdminVariantForm(form AdminVariantForm, create bool, productID string, variantID string) (AdminVariantSaveInput, AdminFieldErrors) {
	errorsByField := AdminFieldErrors{}
	input := AdminVariantSaveInput{
		ID:        normalizeUUID(variantID),
		ProductID: normalizeUUID(productID),
		Name:      strings.TrimSpace(form.Name),
		Slug:      strings.TrimSpace(form.Slug),
		SKU:       strings.TrimSpace(form.SKU),
		IsActive:  form.IsActive,
		IsDefault: form.IsDefault && form.IsActive,
	}
	if !ValidUUID(input.ProductID) || (!create && !ValidUUID(input.ID)) {
		errorsByField.Add("id", "Identificador invalido.")
	}
	if input.Name == "" {
		errorsByField.Add("name", "Informe o nome.")
	}
	if create && form.IsDefault && !form.IsActive {
		errorsByField.Add("is_default", "A variante padrao precisa estar ativa.")
	}
	if input.Slug == "" && create {
		input.Slug = CanonicalSlug(input.Name)
	}
	validateSlugField(errorsByField, "slug", input.Slug, "Informe um slug válido.")
	if strings.TrimSpace(form.PriceBRL) != "" {
		price, err := ParseAdminBRLCents(form.PriceBRL)
		if err != nil {
			errorsByField.Add("price", "Informe um preco em reais ou deixe vazio para herdar.")
		} else {
			input.PriceCents = &price
		}
	}
	sortOrder, err := parseAdminNonNegativeInt(form.SortOrder)
	if err != nil {
		errorsByField.Add("sort_order", "Informe um inteiro maior ou igual a zero.")
	}
	input.SortOrder = sortOrder
	if strings.TrimSpace(form.PrintTimeMinutes) != "" {
		printTime, err := parseAdminPositiveInt(form.PrintTimeMinutes)
		if err != nil {
			errorsByField.Add("print_time_minutes", "Informe minutos positivos ou deixe vazio.")
		} else {
			input.PrintTimeMinutes = &printTime
		}
	}
	input.ShippingProfile = validateAdminShippingProfile(errorsByField, form.UseShipping, form.ShippingWeightG, form.ShippingHeightMM, form.ShippingWidthMM, form.ShippingLengthMM)

	return input, errorsByField
}

func validateAdminRecipeForm(form AdminRecipeForm, productID string, variantID string, componentID string) (AdminRecipeSaveInput, AdminFieldErrors) {
	errorsByField := AdminFieldErrors{}
	input := AdminRecipeSaveInput{
		ID:         normalizeUUID(componentID),
		ProductID:  normalizeUUID(productID),
		VariantID:  normalizeUUID(variantID),
		MaterialID: normalizeUUID(form.MaterialID),
		ColorID:    normalizeUUID(form.ColorID),
		Label:      strings.TrimSpace(form.Label),
	}
	if !ValidUUID(input.ProductID) || !ValidUUID(input.VariantID) || (componentID != "" && !ValidUUID(input.ID)) {
		errorsByField.Add("id", "Identificador invalido.")
	}
	if !ValidUUID(input.MaterialID) {
		errorsByField.Add("material_id", "Selecione um material.")
	}
	if !ValidUUID(input.ColorID) {
		errorsByField.Add("color_id", "Selecione uma cor.")
	}
	weight, err := ParseAdminGramsToMilligrams(form.EstimatedWeight)
	if err != nil {
		errorsByField.Add("estimated_weight", "Informe peso em gramas com ate 3 casas decimais.")
	}
	input.EstimatedWeightMg = weight
	sortOrder, err := parseAdminNonNegativeInt(form.SortOrder)
	if err != nil {
		errorsByField.Add("sort_order", "Informe um inteiro maior ou igual a zero.")
	}
	input.SortOrder = sortOrder

	return input, errorsByField
}

func validateAdminMaterialForm(form AdminMaterialForm, create bool, id string) (AdminMaterialSaveInput, AdminFieldErrors) {
	errorsByField := AdminFieldErrors{}
	input := AdminMaterialSaveInput{
		ID:          normalizeUUID(id),
		Name:        strings.TrimSpace(form.Name),
		Slug:        strings.TrimSpace(form.Slug),
		Description: strings.TrimSpace(form.Description),
		IsActive:    form.IsActive,
	}
	if input.Name == "" {
		errorsByField.Add("name", "Informe o nome.")
	}
	if input.Slug == "" && create {
		input.Slug = CanonicalSlug(input.Name)
	}
	validateSlugField(errorsByField, "slug", input.Slug, "Informe um slug válido.")

	return input, errorsByField
}

func validateAdminColorForm(form AdminColorForm, create bool, id string) (AdminColorSaveInput, AdminFieldErrors) {
	errorsByField := AdminFieldErrors{}
	input := AdminColorSaveInput{
		ID:       normalizeUUID(id),
		Name:     strings.TrimSpace(form.Name),
		Slug:     strings.TrimSpace(form.Slug),
		IsActive: form.IsActive,
	}
	if input.Name == "" {
		errorsByField.Add("name", "Informe o nome.")
	}
	if input.Slug == "" && create {
		input.Slug = CanonicalSlug(input.Name)
	}
	validateSlugField(errorsByField, "slug", input.Slug, "Informe um slug válido.")
	hexColor, err := NormalizeHexColor(form.HexColor)
	if err != nil {
		errorsByField.Add("hex_color", "Use hexadecimal no formato #RRGGBB.")
	}
	input.HexColor = hexColor

	return input, errorsByField
}

func validateAdminBoxForm(form AdminBoxForm, create bool, id string) (AdminBoxSaveInput, AdminFieldErrors) {
	errorsByField := AdminFieldErrors{}
	input := AdminBoxSaveInput{
		ID:       normalizeUUID(id),
		Name:     strings.TrimSpace(form.Name),
		Slug:     strings.TrimSpace(form.Slug),
		IsActive: form.IsActive,
	}
	if input.Name == "" {
		errorsByField.Add("name", "Informe o nome.")
	}
	if input.Slug == "" && create {
		input.Slug = CanonicalSlug(input.Name)
	}
	validateSlugField(errorsByField, "slug", input.Slug, "Informe um slug válido.")
	input.InternalHeightMM = parseBoxPositiveInt(errorsByField, "internal_height_mm", form.InternalHeightMM)
	input.InternalWidthMM = parseBoxPositiveInt(errorsByField, "internal_width_mm", form.InternalWidthMM)
	input.InternalLengthMM = parseBoxPositiveInt(errorsByField, "internal_length_mm", form.InternalLengthMM)
	input.ExternalHeightMM = parseBoxPositiveInt(errorsByField, "external_height_mm", form.ExternalHeightMM)
	input.ExternalWidthMM = parseBoxPositiveInt(errorsByField, "external_width_mm", form.ExternalWidthMM)
	input.ExternalLengthMM = parseBoxPositiveInt(errorsByField, "external_length_mm", form.ExternalLengthMM)
	input.PackagingWeightG = parseBoxPositiveInt(errorsByField, "packaging_weight_g", form.PackagingWeightG)
	sortOrder, err := parseAdminNonNegativeInt(form.SortOrder)
	if err != nil {
		errorsByField.Add("sort_order", "Informe um inteiro maior ou igual a zero.")
	}
	input.SortOrder = sortOrder
	if input.ExternalHeightMM > 0 && input.InternalHeightMM > 0 && input.ExternalHeightMM < input.InternalHeightMM {
		errorsByField.Add("external_height_mm", "Altura externa deve ser maior ou igual a interna.")
	}
	if input.ExternalWidthMM > 0 && input.InternalWidthMM > 0 && input.ExternalWidthMM < input.InternalWidthMM {
		errorsByField.Add("external_width_mm", "Largura externa deve ser maior ou igual a interna.")
	}
	if input.ExternalLengthMM > 0 && input.InternalLengthMM > 0 && input.ExternalLengthMM < input.InternalLengthMM {
		errorsByField.Add("external_length_mm", "Comprimento externo deve ser maior ou igual ao interno.")
	}

	return input, errorsByField
}

func validateSlugField(errorsByField AdminFieldErrors, field string, slug string, message string) {
	if slug == "" || !products.ValidSlug(slug) {
		errorsByField.Add(field, message)
	}
}

func validateAdminShippingProfile(errorsByField AdminFieldErrors, enabled bool, weight string, height string, width string, length string) *AdminShippingProfile {
	values := []string{strings.TrimSpace(weight), strings.TrimSpace(height), strings.TrimSpace(width), strings.TrimSpace(length)}
	anyFilled := false
	for _, value := range values {
		if value != "" {
			anyFilled = true
		}
	}
	if !enabled {
		if anyFilled {
			errorsByField.Add("shipping_profile", "Marque o uso do perfil logistico ou limpe os campos.")
		}
		return nil
	}

	profile := &AdminShippingProfile{}
	var err error
	profile.WeightG, err = parseAdminPositiveInt64(weight)
	if err != nil {
		errorsByField.Add("shipping_weight_g", "Informe peso positivo em gramas.")
	}
	profile.HeightMM, err = parseAdminPositiveInt(height)
	if err != nil {
		errorsByField.Add("shipping_height_mm", "Informe altura positiva em mm.")
	}
	profile.WidthMM, err = parseAdminPositiveInt(width)
	if err != nil {
		errorsByField.Add("shipping_width_mm", "Informe largura positiva em mm.")
	}
	profile.LengthMM, err = parseAdminPositiveInt(length)
	if err != nil {
		errorsByField.Add("shipping_length_mm", "Informe comprimento positivo em mm.")
	}

	return profile
}

func parseBoxPositiveInt(errorsByField AdminFieldErrors, field string, value string) int {
	parsed, err := parseAdminPositiveInt(value)
	if err != nil {
		errorsByField.Add(field, "Informe um valor inteiro positivo.")
	}

	return parsed
}

func parseAdminPositiveInt(value string) (int, error) {
	parsed, err := parseAdminPositiveInt64(value)
	if err != nil || parsed > int64(math.MaxInt32) {
		return 0, ErrValidation
	}

	return int(parsed), nil
}

func parseAdminNonNegativeInt(value string) (int, error) {
	parsed, err := parseAdminNonNegativeInt64(value)
	if err != nil || parsed > int64(math.MaxInt32) {
		return 0, ErrValidation
	}

	return int(parsed), nil
}

func parseAdminPositiveInt64(value string) (int64, error) {
	parsed, err := parseAdminNonNegativeInt64(value)
	if err != nil || parsed <= 0 {
		return 0, ErrValidation
	}

	return parsed, nil
}

func parseAdminNonNegativeInt64(value string) (int64, error) {
	value = strings.TrimSpace(value)
	if value == "" || strings.HasPrefix(value, "-") || strings.HasPrefix(value, "+") || !adminASCIIDigits(value) {
		return 0, ErrValidation
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed < 0 {
		return 0, ErrValidation
	}

	return parsed, nil
}

func parseScaledAdminDecimal(value string, scale int64, maxFractionDigits int, allowZero bool) (int64, error) {
	value = strings.TrimSpace(value)
	if value == "" || strings.HasPrefix(value, "-") || strings.HasPrefix(value, "+") {
		return 0, ErrValidation
	}
	if strings.ContainsAny(value, "eE") || (strings.Contains(value, ",") && strings.Contains(value, ".")) {
		return 0, ErrValidation
	}

	separator := ""
	if strings.Contains(value, ",") {
		separator = ","
	} else if strings.Contains(value, ".") {
		separator = "."
	}
	integerPart := value
	fractionPart := ""
	if separator != "" {
		parts := strings.Split(value, separator)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return 0, ErrValidation
		}
		integerPart = parts[0]
		fractionPart = parts[1]
	}
	if !adminASCIIDigits(integerPart) || (fractionPart != "" && !adminASCIIDigits(fractionPart)) {
		return 0, ErrValidation
	}
	if len(fractionPart) > maxFractionDigits {
		return 0, ErrValidation
	}

	integer, err := strconv.ParseInt(integerPart, 10, 64)
	if err != nil || integer < 0 || integer > math.MaxInt64/scale {
		return 0, ErrValidation
	}
	scaled := integer * scale
	if fractionPart != "" {
		padded := fractionPart + strings.Repeat("0", maxFractionDigits-len(fractionPart))
		fraction, err := strconv.ParseInt(padded, 10, 64)
		if err != nil {
			return 0, ErrValidation
		}
		if scaled > math.MaxInt64-fraction {
			return 0, ErrValidation
		}
		scaled += fraction
	}
	if scaled == 0 && !allowZero {
		return 0, ErrValidation
	}

	return scaled, nil
}

func adminASCIIDigits(value string) bool {
	if value == "" {
		return false
	}
	for _, char := range value {
		if char < '0' || char > '9' {
			return false
		}
	}

	return true
}

func canonicalSlugRune(char rune) byte {
	if char >= 'a' && char <= 'z' {
		return byte(char)
	}
	if char >= '0' && char <= '9' {
		return byte(char)
	}
	if char > unicode.MaxASCII {
		switch char {
		case 'á', 'à', 'â', 'ã', 'ä':
			return 'a'
		case 'é', 'è', 'ê', 'ë':
			return 'e'
		case 'í', 'ì', 'î', 'ï':
			return 'i'
		case 'ó', 'ò', 'ô', 'õ', 'ö':
			return 'o'
		case 'ú', 'ù', 'û', 'ü':
			return 'u'
		case 'ç':
			return 'c'
		case 'ñ':
			return 'n'
		}
	}

	return 0
}

func catalogSaveErrors(err error) AdminFieldErrors {
	errorsByField := AdminFieldErrors{}
	switch {
	case errors.Is(err, ErrDuplicateSlug):
		errorsByField.Add("slug", "Este slug ja esta em uso. Ajuste e tente novamente.")
	case errors.Is(err, ErrDuplicateSKU):
		errorsByField.Add("sku", "Este SKU ja esta em uso. Ajuste e tente novamente.")
	}

	return errorsByField
}

func leftPadAdmin2(value int64) string {
	if value < 10 {
		return "0" + strconv.FormatInt(value, 10)
	}

	return strconv.FormatInt(value, 10)
}
